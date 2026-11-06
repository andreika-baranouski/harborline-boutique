package main

import (
	"bytes"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	pb "github.com/GoogleCloudPlatform/microservices-demo/src/frontend/genproto"
)

func usd(u int64, n int32) *pb.Money { return &pb.Money{CurrencyCode: "USD", Units: u, Nanos: n} }

func sampleOrder() *pb.OrderResult {
	return &pb.OrderResult{
		OrderId:            "order-1",
		ShippingTrackingId: "track-1",
		ShippingCost:       usd(8, 990000000),
		Items: []*pb.OrderItem{
			{Item: &pb.CartItem{ProductId: "OLJCESPC7Z", Quantity: 2}, Cost: usd(19, 990000000)},
		},
	}
}

func TestOrderTotalPaid(t *testing.T) {
	order := sampleOrder()
	got := orderTotalPaid(order)
	if got.GetUnits() != 48 || got.GetNanos() != 970000000 {
		t.Errorf("fallback orderTotalPaid() = %v, want 48.97", &got)
	}

	order.TotalPaid = usd(44, 970000000)
	got = orderTotalPaid(order)
	if got.GetUnits() != 44 || got.GetNanos() != 970000000 {
		t.Errorf("orderTotalPaid() = %v, want TotalPaid 44.97", &got)
	}
}

func renderOrder(t *testing.T, promo *pb.PromoResult) string {
	t.Helper()
	order := sampleOrder()
	order.Promo = promo
	total := orderTotalPaid(order)
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, "order", map[string]interface{}{
		"order":      order,
		"total_paid": &total,
		"promo":      promo,
	}); err != nil {
		t.Fatalf("ExecuteTemplate(order) error = %v", err)
	}
	return strings.Join(strings.Fields(buf.String()), " ")
}

func TestOrderTemplatePromo(t *testing.T) {
	tests := []struct {
		name    string
		promo   *pb.PromoResult
		want    []string
		notWant []string
	}{
		{
			name:  "applied",
			promo: &pb.PromoResult{Code: "WELCOME10", Status: pb.PromoResult_APPLIED, Percent: 10, Discount: usd(2, 0)},
			want:  []string{"WELCOME10 </div> <div class=\"col-6 pr-md-0 text-right\"> &minus;$2.00"},
		},
		{
			name:  "unknown",
			promo: &pb.PromoResult{Code: "NOPE", Status: pb.PromoResult_UNKNOWN},
			want:  []string{"Promo code NOPE is not valid. Your order was placed without a discount."},
		},
		{
			name:  "expired",
			promo: &pb.PromoResult{Code: "HARBOR25", Status: pb.PromoResult_EXPIRED},
			want:  []string{"Promo code HARBOR25 has expired. Your order was placed without a discount."},
		},
		{
			name:  "not yet active",
			promo: &pb.PromoResult{Code: "HARBOR25", Status: pb.PromoResult_NOT_YET_ACTIVE},
			want:  []string{"Promo code HARBOR25 is not valid yet."},
		},
		{
			name:    "no promo",
			promo:   nil,
			notWant: []string{"order-promo-row", "order-promo-notice", "Promo code"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderOrder(t, tt.promo)
			for _, w := range tt.want {
				if !strings.Contains(got, w) {
					t.Errorf("order page does not contain %q", w)
				}
			}
			for _, w := range tt.notWant {
				if strings.Contains(got, w) {
					t.Errorf("order page contains %q, want absent", w)
				}
			}
		})
	}
}

func TestCartTemplatePromoInput(t *testing.T) {
	for _, enabled := range []bool{true, false} {
		var buf bytes.Buffer
		if err := templates.ExecuteTemplate(&buf, "cart", map[string]interface{}{
			"cart_size": 1,
			"items": []struct {
				Item     *pb.Product
				Quantity int32
				Price    *pb.Money
			}{{Item: &pb.Product{Id: "OLJCESPC7Z", Name: "Sunglasses"}, Quantity: 1, Price: usd(19, 990000000)}},
			"shipping_cost": usd(8, 990000000),
			"total_cost":    usd(28, 980000000),
			"promo_enabled": enabled,
		}); err != nil {
			t.Fatalf("ExecuteTemplate(cart) error = %v", err)
		}
		got := buf.String()
		if has := strings.Contains(got, `name="promo_code" form="checkout-form"`); has != enabled {
			t.Errorf("promo_enabled=%v: promo input rendered = %v", enabled, has)
		}
		if !strings.Contains(got, `<form id="checkout-form"`) {
			t.Errorf("checkout form has no id=checkout-form")
		}
	}
}

func TestPromoCodeFromForm(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		enabled bool
		want    string
	}{
		{"normalised", "  welcome10 ", true, "WELCOME10"},
		{"empty", "", true, ""},
		{"disabled drops code", "WELCOME10", false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := url.Values{"promo_code": {tt.value}}
			r := httptest.NewRequest("POST", "/cart/checkout", strings.NewReader(form.Encode()))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			if got := promoCodeFromForm(r, tt.enabled); got != tt.want {
				t.Errorf("promoCodeFromForm(%q, %v) = %q, want %q", tt.value, tt.enabled, got, tt.want)
			}
		})
	}
}
