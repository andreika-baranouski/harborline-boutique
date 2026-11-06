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
	"bytes"
	"strings"
	"testing"

	pb "github.com/GoogleCloudPlatform/microservices-demo/src/frontend/genproto"
)

// renderOrderPage executes the order-complete template for order.
func renderOrderPage(t *testing.T, order *pb.OrderResult) string {
	t.Helper()
	var buf bytes.Buffer
	err := templates.ExecuteTemplate(&buf, "order", map[string]interface{}{
		"order":      order,
		"total_paid": &pb.Money{CurrencyCode: "USD", Units: 28, Nanos: 980000000},
	})
	if err != nil {
		t.Fatalf("render order template: %v", err)
	}
	return buf.String()
}

func TestOrderPageShowsEstimatedDelivery(t *testing.T) {
	page := renderOrderPage(t, &pb.OrderResult{OrderId: "o-1", ShippingTrackingId: "AB-1-2", EstimatedDelivery: "2026-11-12"})
	if !strings.Contains(page, "Estimated delivery") || !strings.Contains(page, "Thursday, November 12, 2026") {
		t.Fatalf("order page has no estimated delivery row:\n%s", page)
	}
	if strings.Index(page, "AB-1-2") > strings.Index(page, "Estimated delivery") {
		t.Errorf("estimated delivery row should follow the tracking row")
	}
}

func TestOrderPageOmitsMissingEstimate(t *testing.T) {
	for _, est := range []string{"", "not-a-date"} {
		page := renderOrderPage(t, &pb.OrderResult{OrderId: "o-1", ShippingTrackingId: "AB-1-2", EstimatedDelivery: est})
		if strings.Contains(page, "Estimated delivery") {
			t.Errorf("order page shows an estimated delivery row for %q", est)
		}
	}
}
