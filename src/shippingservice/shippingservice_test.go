// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"regexp"
	"testing"
	"time"

	"golang.org/x/net/context"

	pb "github.com/GoogleCloudPlatform/microservices-demo/src/shippingservice/genproto"
)

// TestGetQuote is a basic check on the GetQuote RPC service.
func TestGetQuote(t *testing.T) {
	s := server{}

	// A basic test case to test logic and protobuf interactions.
	req := &pb.GetQuoteRequest{
		Address: &pb.Address{
			StreetAddress: "Muffin Man",
			City:          "London",
			State:         "",
			Country:       "England",
		},
		Items: []*pb.CartItem{
			{
				ProductId: "23",
				Quantity:  1,
			},
			{
				ProductId: "46",
				Quantity:  3,
			},
		},
	}

	res, err := s.GetQuote(context.Background(), req)
	if err != nil {
		t.Errorf("TestGetQuote (%v) failed", err)
	}
	if res.CostUsd.GetUnits() != 8 || res.CostUsd.GetNanos() != 990000000 {
		t.Errorf("TestGetQuote: Quote value '%d.%d' does not match expected '8.990000000'", res.CostUsd.GetUnits(), res.CostUsd.GetNanos())
	}
}

// TestGetQuoteEmptyCart verifies that an empty cart returns a zero quote.
func TestGetQuoteEmptyCart(t *testing.T) {
	s := server{}

	req := &pb.GetQuoteRequest{
		Address: &pb.Address{
			StreetAddress: "221B Baker Street",
			City:          "London",
			State:         "",
			Country:       "England",
		},
		Items: []*pb.CartItem{},
	}

	res, err := s.GetQuote(context.Background(), req)
	if err != nil {
		t.Errorf("TestGetQuoteEmptyCart (%v) failed", err)
	}
	if res.CostUsd.GetUnits() != 0 || res.CostUsd.GetNanos() != 0 {
		t.Errorf("TestGetQuoteEmptyCart: expected zero quote for empty cart, got '%d.%d'", res.CostUsd.GetUnits(), res.CostUsd.GetNanos())
	}
}

// TestShipOrder is a basic check on the ShipOrder RPC service.
func TestShipOrder(t *testing.T) {
	s := server{}

	// A basic test case to test logic and protobuf interactions.
	req := &pb.ShipOrderRequest{
		Address: &pb.Address{
			StreetAddress: "Muffin Man",
			City:          "London",
			State:         "",
			Country:       "England",
		},
		Items: []*pb.CartItem{
			{
				ProductId: "23",
				Quantity:  1,
			},
			{
				ProductId: "46",
				Quantity:  3,
			},
		},
	}

	res, err := s.ShipOrder(context.Background(), req)
	if err != nil {
		t.Errorf("TestShipOrder (%v) failed", err)
	}
	if len(res.TrackingId) != 18 {
		t.Errorf("TestShipOrder: Tracking ID is malformed - has %d characters, %d expected", len(res.TrackingId), 18)
	}
}

// TestTrackingIdFormat verifies the tracking ID matches the expected pattern.
func TestTrackingIdFormat(t *testing.T) {
	pattern := regexp.MustCompile(`^[A-Z]{2}-\d+-\d+$`)

	for i := 0; i < 20; i++ {
		id := CreateTrackingId("test-salt-value")
		if !pattern.MatchString(id) {
			t.Errorf("CreateTrackingId: '%s' does not match expected pattern '[A-Z]{2}-\\d+-\\d+'", id)
		}
	}
}

// TestTrackingIdUniqueness checks that generated IDs are not all identical.
func TestTrackingIdUniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 50; i++ {
		id := CreateTrackingId("same-salt")
		seen[id] = true
	}
	if len(seen) < 2 {
		t.Errorf("CreateTrackingId: expected unique IDs but got %d distinct values out of 50", len(seen))
	}
}

// TestCreateQuoteFromFloat verifies quote creation from float values.
func TestCreateQuoteFromFloat(t *testing.T) {
	tests := []struct {
		name    string
		value   float64
		dollars uint32
		cents   uint32
	}{
		{"zero", 0.0, 0, 0},
		{"whole dollars", 10.0, 10, 0},
		{"with cents", 8.99, 8, 99},
		{"small value", 0.50, 0, 50},
		{"large value", 100.01, 100, 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			q := CreateQuoteFromFloat(tc.value)
			if q.Dollars != tc.dollars || q.Cents != tc.cents {
				t.Errorf("CreateQuoteFromFloat(%v) = $%d.%d, want $%d.%d",
					tc.value, q.Dollars, q.Cents, tc.dollars, tc.cents)
			}
		})
	}
}

// TestCreateQuoteFromCount verifies count-based quote generation.
func TestCreateQuoteFromCount(t *testing.T) {
	zeroQuote := CreateQuoteFromCount(0, "US")
	if zeroQuote.Dollars != 0 || zeroQuote.Cents != 0 {
		t.Errorf("CreateQuoteFromCount(0) = %s, want $0.0", zeroQuote)
	}

	nonZeroQuote := CreateQuoteFromCount(5, "US")
	if nonZeroQuote.Dollars == 0 && nonZeroQuote.Cents == 0 {
		t.Error("CreateQuoteFromCount(5) returned zero, expected a non-zero quote")
	}
}

// TestCreateQuoteFromCountDefaultRate verifies that destinations without a
// regional rate (including an empty country) get the default rate.
func TestCreateQuoteFromCountDefaultRate(t *testing.T) {
	for _, country := range []string{"US", "DE", "Germany", "Canada", "England", ""} {
		q := CreateQuoteFromCount(1, country)
		if q.Dollars != 8 || q.Cents != 99 {
			t.Errorf("CreateQuoteFromCount(1, %q) = %s, want $8.99", country, q)
		}
	}
}

// TestNormalizeCountry verifies that free-text countries are trimmed, upper-cased and de-aliased.
func TestNormalizeCountry(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"US", "US"},
		{"us", "US"},
		{"  US ", "US"},
		{"USA", "US"},
		{"usa", "US"},
		{"U.S.", "US"},
		{"u.s.", "US"},
		{"United States", "US"},
		{"united states", "US"},
		{" United States of America ", "US"},
		{"Germany", "GERMANY"},
		{" de ", "DE"},
		{"", ""},
	}
	for _, tc := range tests {
		if got := normalizeCountry(tc.in); got != tc.want {
			t.Errorf("normalizeCountry(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestCreateQuoteFromCountRegionalRateVariants verifies that a regional rate
// applies to every way customers type the country, and others get the default.
func TestCreateQuoteFromCountRegionalRateVariants(t *testing.T) {
	orig := regionalRates["US"]
	regionalRates["US"] = 5.49
	defer func() { regionalRates["US"] = orig }()

	for _, country := range []string{"US", "us", " US ", "USA", "U.S.", "United States", "united states", "United States of America"} {
		if q := CreateQuoteFromCount(1, country); q.Dollars != 5 || q.Cents != 49 {
			t.Errorf("CreateQuoteFromCount(1, %q) = %s, want $5.49", country, q)
		}
	}
	for _, country := range []string{"DE", "Germany", "", "U.S.A"} {
		if q := CreateQuoteFromCount(1, country); q.Dollars != 8 || q.Cents != 99 {
			t.Errorf("CreateQuoteFromCount(1, %q) = %s, want $8.99", country, q)
		}
	}
}

// TestGetQuoteNoAddress verifies that a request without an address gets the default rate.
func TestGetQuoteNoAddress(t *testing.T) {
	s := server{}
	req := &pb.GetQuoteRequest{
		Items: []*pb.CartItem{{ProductId: "23", Quantity: 1}},
	}
	res, err := s.GetQuote(context.Background(), req)
	if err != nil {
		t.Fatalf("TestGetQuoteNoAddress (%v) failed", err)
	}
	if res.CostUsd.GetUnits() != 8 || res.CostUsd.GetNanos() != 990000000 {
		t.Errorf("TestGetQuoteNoAddress: Quote value '%d.%d' does not match expected '8.990000000'", res.CostUsd.GetUnits(), res.CostUsd.GetNanos())
	}
}

// TestGetRandomLetterCode verifies the output is a valid uppercase letter.
func TestGetRandomLetterCode(t *testing.T) {
	for i := 0; i < 100; i++ {
		code := getRandomLetterCode()
		if code < 65 || code > 90 {
			t.Errorf("getRandomLetterCode: got %d (%c), expected range 65-90 (A-Z)", code, code)
		}
	}
}

// TestGetRandomNumber verifies the output has the correct number of digits.
func TestGetRandomNumber(t *testing.T) {
	for _, digits := range []int{1, 3, 5, 7, 10} {
		result := getRandomNumber(digits)
		if len(result) != digits {
			t.Errorf("getRandomNumber(%d) = '%s' (len %d), expected length %d",
				digits, result, len(result), digits)
		}
	}
}

// TestQuoteString verifies the string representation of a Quote.
func TestQuoteString(t *testing.T) {
	q := Quote{Dollars: 8, Cents: 99}
	expected := "$8.99"
	if q.String() != expected {
		t.Errorf("Quote.String() = '%s', want '%s'", q.String(), expected)
	}
}

// shipDay returns a fixed UTC ship date for the delivery-estimate tests.
func shipDay(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 15, 30, 0, 0, time.UTC)
}

// estimateServer returns a server with delivery estimates enabled and a fixed clock.
func estimateServer(now time.Time) *server {
	return &server{estimatesEnabled: true, now: func() time.Time { return now }}
}

// TestEstimateDelivery covers the business-day rule: US 3 days, everywhere else 7, weekends skipped.
func TestEstimateDelivery(t *testing.T) {
	tests := []struct {
		name    string
		ship    time.Time
		country string
		want    string
	}{
		{"US Monday to Thursday", shipDay(2026, time.November, 9), "US", "2026-11-12"},
		{"usa alias", shipDay(2026, time.November, 9), "usa", "2026-11-12"},
		{"United States alias", shipDay(2026, time.November, 9), " United States ", "2026-11-12"},
		{"non-US 7 business days", shipDay(2026, time.November, 9), "DE", "2026-11-18"},
		{"unknown country uses default", shipDay(2026, time.November, 9), "", "2026-11-18"},
		{"US Thursday skips weekend", shipDay(2026, time.November, 12), "US", "2026-11-17"},
		{"US Saturday counts from Monday", shipDay(2026, time.November, 14), "US", "2026-11-18"},
		{"US Sunday counts from Monday", shipDay(2026, time.November, 15), "US", "2026-11-18"},
		{"US year boundary", shipDay(2026, time.December, 30), "US", "2027-01-04"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := EstimateDelivery(tc.ship, tc.country).Format(deliveryDateLayout)
			if got != tc.want {
				t.Errorf("EstimateDelivery(%s, %q) = %s, want %s", tc.ship.Format(time.RFC3339), tc.country, got, tc.want)
			}
		})
	}
}

// TestEstimateDeliveryUsesUTCDate checks that the ship date is the UTC calendar date of the clock.
func TestEstimateDeliveryUsesUTCDate(t *testing.T) {
	// Sunday 2026-11-08 20:00 in New York is already Monday 2026-11-09 in UTC.
	ny := time.FixedZone("EST", -5*60*60)
	got := EstimateDelivery(time.Date(2026, time.November, 8, 20, 0, 0, 0, ny), "US").Format(deliveryDateLayout)
	if got != "2026-11-12" {
		t.Errorf("EstimateDelivery = %s, want 2026-11-12", got)
	}
}

// TestEstimateDeliveryTableDriven checks that a changed transitDays entry is used without code changes.
func TestEstimateDeliveryTableDriven(t *testing.T) {
	old := transitDays["US"]
	t.Cleanup(func() { transitDays["US"] = old })
	transitDays["US"] = 2

	got := EstimateDelivery(shipDay(2026, time.November, 9), "US").Format(deliveryDateLayout)
	if got != "2026-11-11" {
		t.Errorf("EstimateDelivery with US=2 = %s, want 2026-11-11", got)
	}
}

// TestGetQuoteEstimatedDelivery checks that GetQuote carries the estimate and leaves the cost unchanged.
func TestGetQuoteEstimatedDelivery(t *testing.T) {
	s := estimateServer(shipDay(2026, time.November, 9))
	req := &pb.GetQuoteRequest{
		Address: &pb.Address{Country: "US"},
		Items:   []*pb.CartItem{{ProductId: "23", Quantity: 1}},
	}

	res, err := s.GetQuote(context.Background(), req)
	if err != nil {
		t.Fatalf("GetQuote failed: %v", err)
	}
	if res.CostUsd.GetUnits() != 8 || res.CostUsd.GetNanos() != 990000000 {
		t.Errorf("GetQuote cost = %d.%d, want 8.990000000", res.CostUsd.GetUnits(), res.CostUsd.GetNanos())
	}
	if res.GetEstimatedDelivery() != "2026-11-12" {
		t.Errorf("GetQuote estimated_delivery = %q, want 2026-11-12", res.GetEstimatedDelivery())
	}
}

// TestGetQuoteEstimatedDeliveryNoAddress checks that the cart-page quote (no address) uses the default transit time.
func TestGetQuoteEstimatedDeliveryNoAddress(t *testing.T) {
	s := estimateServer(shipDay(2026, time.November, 9))
	res, err := s.GetQuote(context.Background(), &pb.GetQuoteRequest{Items: []*pb.CartItem{{ProductId: "23", Quantity: 1}}})
	if err != nil {
		t.Fatalf("GetQuote failed: %v", err)
	}
	if res.GetEstimatedDelivery() != "2026-11-18" {
		t.Errorf("GetQuote estimated_delivery = %q, want 2026-11-18", res.GetEstimatedDelivery())
	}
}

// TestShipOrderEstimatedDelivery checks that ShipOrder returns both the tracking id and the estimate.
func TestShipOrderEstimatedDelivery(t *testing.T) {
	s := estimateServer(shipDay(2026, time.November, 9))
	req := &pb.ShipOrderRequest{
		Address: &pb.Address{StreetAddress: "1600 Amphitheatre Pkwy", City: "Mountain View", State: "CA", Country: "United States"},
		Items:   []*pb.CartItem{{ProductId: "23", Quantity: 1}},
	}

	res, err := s.ShipOrder(context.Background(), req)
	if err != nil {
		t.Fatalf("ShipOrder failed: %v", err)
	}
	if !regexp.MustCompile(`^[A-Z]{2}-\d+-\d+$`).MatchString(res.GetTrackingId()) {
		t.Errorf("ShipOrder tracking_id = %q does not match the expected format", res.GetTrackingId())
	}
	if res.GetEstimatedDelivery() != "2026-11-12" {
		t.Errorf("ShipOrder estimated_delivery = %q, want 2026-11-12", res.GetEstimatedDelivery())
	}
}

// TestEstimateDisabled checks that both RPCs leave estimated_delivery empty when the flag is off.
func TestEstimateDisabled(t *testing.T) {
	s := &server{now: func() time.Time { return shipDay(2026, time.November, 9) }}
	addr := &pb.Address{StreetAddress: "1 Main St", City: "Springfield", State: "IL", Country: "US"}
	items := []*pb.CartItem{{ProductId: "23", Quantity: 1}}

	quote, err := s.GetQuote(context.Background(), &pb.GetQuoteRequest{Address: addr, Items: items})
	if err != nil {
		t.Fatalf("GetQuote failed: %v", err)
	}
	if quote.GetEstimatedDelivery() != "" {
		t.Errorf("GetQuote estimated_delivery = %q, want empty", quote.GetEstimatedDelivery())
	}
	ship, err := s.ShipOrder(context.Background(), &pb.ShipOrderRequest{Address: addr, Items: items})
	if err != nil {
		t.Fatalf("ShipOrder failed: %v", err)
	}
	if ship.GetEstimatedDelivery() != "" {
		t.Errorf("ShipOrder estimated_delivery = %q, want empty", ship.GetEstimatedDelivery())
	}
}
