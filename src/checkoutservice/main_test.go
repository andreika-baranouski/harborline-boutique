package main

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	pb "github.com/GoogleCloudPlatform/microservices-demo/src/checkoutservice/genproto"
	money "github.com/GoogleCloudPlatform/microservices-demo/src/checkoutservice/money"
)

func usd(u int64, n int32) pb.Money { return pb.Money{CurrencyCode: "USD", Units: u, Nanos: n} }

func item(cost pb.Money, qty int32) *pb.OrderItem {
	return &pb.OrderItem{Item: &pb.CartItem{ProductId: "OLJCESPC7Z", Quantity: qty}, Cost: &cost}
}

func applied(pct uint32) *pb.PromoResult {
	return &pb.PromoResult{Code: "X", Status: pb.PromoResult_APPLIED, Percent: pct}
}

func TestComputeTotals(t *testing.T) {
	shipping := usd(8, 990000000)
	tests := []struct {
		name         string
		items        []*pb.OrderItem
		promo        *pb.PromoResult
		wantTotal    pb.Money
		wantDiscount pb.Money
	}{
		{"WELCOME10 on one 19.99 item", []*pb.OrderItem{item(usd(19, 990000000), 1)}, applied(10), usd(26, 980000000), usd(2, 0)},
		{"quantity > 1", []*pb.OrderItem{item(usd(19, 990000000), 3)}, applied(10), usd(62, 960000000), usd(6, 0)},
		{"HARBOR25 over two items", []*pb.OrderItem{item(usd(19, 990000000), 1), item(usd(10, 0), 2)}, applied(25), usd(38, 980000000), usd(10, 0)},
		{"100% leaves shipping", []*pb.OrderItem{item(usd(19, 990000000), 1)}, applied(100), shipping, usd(19, 990000000)},
		{"no promo", []*pb.OrderItem{item(usd(19, 990000000), 1)}, nil, usd(28, 980000000), usd(0, 0)},
		{"unknown code", []*pb.OrderItem{item(usd(19, 990000000), 1)}, &pb.PromoResult{Code: "NOPE", Status: pb.PromoResult_UNKNOWN}, usd(28, 980000000), usd(0, 0)},
		{"expired code", []*pb.OrderItem{item(usd(19, 990000000), 1)}, &pb.PromoResult{Code: "HARBOR25", Status: pb.PromoResult_EXPIRED, Percent: 25}, usd(28, 980000000), usd(0, 0)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			total, discount := computeTotals("USD", tt.items, shipping, tt.promo)
			if !money.AreEquals(total, tt.wantTotal) {
				t.Errorf("total = %v, want %v", &total, &tt.wantTotal)
			}
			if !money.AreEquals(discount, tt.wantDiscount) {
				t.Errorf("discount = %v, want %v", &discount, &tt.wantDiscount)
			}
		})
	}
}

func TestComputeTotalsNonUSD(t *testing.T) {
	eur := func(u int64, n int32) pb.Money { return pb.Money{CurrencyCode: "EUR", Units: u, Nanos: n} }
	cost := eur(18, 393000000)
	items := []*pb.OrderItem{{Item: &pb.CartItem{Quantity: 1}, Cost: &cost}}
	total, discount := computeTotals("EUR", items, eur(8, 0), applied(10))
	if !money.AreEquals(discount, eur(1, 840000000)) {
		t.Errorf("discount = %v, want 1.84 EUR", &discount)
	}
	// 18.393 - 1.84 + 8 = 24.553: only the discount is rounded.
	if !money.AreEquals(total, eur(24, 553000000)) {
		t.Errorf("total = %v, want 24.553 EUR", &total)
	}
}

// fakeShipping answers ShipOrder with a fixed response.
type fakeShipping struct {
	pb.UnimplementedShippingServiceServer
	resp *pb.ShipOrderResponse
}

func (f *fakeShipping) ShipOrder(ctx context.Context, in *pb.ShipOrderRequest) (*pb.ShipOrderResponse, error) {
	return f.resp, nil
}

// dialFakeShipping serves fake over an in-memory listener and returns a client connection to it.
func dialFakeShipping(t *testing.T, fake *fakeShipping) *grpc.ClientConn {
	t.Helper()
	lis := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer()
	pb.RegisterShippingServiceServer(srv, fake)
	go srv.Serve(lis)
	t.Cleanup(srv.Stop)

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return lis.DialContext(ctx) }),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("dial fake shipping: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

// TestShipOrderReturnsEstimate checks that shipOrder keeps both the tracking id and the estimated delivery date.
func TestShipOrderReturnsEstimate(t *testing.T) {
	cs := &checkoutService{shippingSvcConn: dialFakeShipping(t, &fakeShipping{
		resp: &pb.ShipOrderResponse{TrackingId: "AB-1-2", EstimatedDelivery: "2026-11-12"},
	})}

	got, err := cs.shipOrder(context.Background(), &pb.Address{Country: "US"}, nil)
	if err != nil {
		t.Fatalf("shipOrder failed: %v", err)
	}
	if got.GetTrackingId() != "AB-1-2" || got.GetEstimatedDelivery() != "2026-11-12" {
		t.Errorf("shipOrder = {%q, %q}, want {AB-1-2, 2026-11-12}", got.GetTrackingId(), got.GetEstimatedDelivery())
	}
}

// TestNewOrderResult checks that the shipment's tracking id and estimate are copied into the order unchanged,
// next to the amount charged and the promo outcome.
func TestNewOrderResult(t *testing.T) {
	addr := &pb.Address{Country: "US"}
	prep := orderPrep{
		orderItems:            []*pb.OrderItem{{Item: &pb.CartItem{ProductId: "OLJCESPC7Z", Quantity: 1}}},
		shippingCostLocalized: &pb.Money{CurrencyCode: "USD", Units: 8, Nanos: 990000000},
	}
	total := usd(26, 980000000)
	promoRes := applied(10)

	tests := []struct {
		name     string
		shipment *pb.ShipOrderResponse
		want     string
	}{
		{"with estimate", &pb.ShipOrderResponse{TrackingId: "AB-1-2", EstimatedDelivery: "2026-11-12"}, "2026-11-12"},
		{"without estimate", &pb.ShipOrderResponse{TrackingId: "AB-1-2"}, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := newOrderResult("o-1", tc.shipment, prep, addr, &total, promoRes)
			if got.GetEstimatedDelivery() != tc.want {
				t.Errorf("EstimatedDelivery = %q, want %q", got.GetEstimatedDelivery(), tc.want)
			}
			if got.GetShippingTrackingId() != "AB-1-2" {
				t.Errorf("ShippingTrackingId = %q, want AB-1-2", got.GetShippingTrackingId())
			}
			if got.GetOrderId() != "o-1" || got.GetShippingAddress() != addr || got.GetShippingCost() != prep.shippingCostLocalized || len(got.GetItems()) != 1 {
				t.Errorf("newOrderResult did not carry the order fields: %v", got)
			}
			if got.GetTotalPaid() != &total || got.GetPromo() != promoRes {
				t.Errorf("newOrderResult did not carry TotalPaid/Promo: %v", got)
			}
		})
	}
}
