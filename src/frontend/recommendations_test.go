package main

import (
	"bytes"
	"context"
	"net"
	"strings"
	"sync"
	"testing"

	pb "github.com/GoogleCloudPlatform/microservices-demo/src/frontend/genproto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

func TestEnvFlagEnabled(t *testing.T) {
	tests := []struct {
		in          string
		wantEnabled bool
		wantOK      bool
	}{
		{"", true, true},
		{"1", true, true},
		{"true", true, true},
		{"TRUE", true, true},
		{" on ", true, true},
		{"yes", true, true},
		{"0", false, true},
		{"false", false, true},
		{"False", false, true},
		{" FALSE ", false, true},
		{"off", false, true},
		{"no", false, true},
		{"flase", true, false},
	}
	for _, tt := range tests {
		enabled, ok := envFlagEnabled(tt.in, true)
		if enabled != tt.wantEnabled || ok != tt.wantOK {
			t.Errorf("envFlagEnabled(%q, true) = (%v, %v), want (%v, %v)", tt.in, enabled, ok, tt.wantEnabled, tt.wantOK)
		}
	}
}

type fakeRecommendationService struct {
	pb.UnimplementedRecommendationServiceServer
	mu    sync.Mutex
	calls []*pb.ListRecommendationsRequest
	ids   []string
}

func (f *fakeRecommendationService) ListRecommendations(_ context.Context, req *pb.ListRecommendationsRequest) (*pb.ListRecommendationsResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, req)
	return &pb.ListRecommendationsResponse{ProductIds: f.ids}, nil
}

func (f *fakeRecommendationService) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

type fakeProductCatalogService struct {
	pb.UnimplementedProductCatalogServiceServer
}

func (fakeProductCatalogService) GetProduct(_ context.Context, req *pb.GetProductRequest) (*pb.Product, error) {
	return &pb.Product{Id: req.GetId(), Name: "product " + req.GetId()}, nil
}

// newBufconnServer serves the fakes in-process and returns a client connection to them.
func newBufconnServer(t *testing.T, rec *fakeRecommendationService) *grpc.ClientConn {
	t.Helper()
	lis := bufconn.Listen(1 << 20)
	srv := grpc.NewServer()
	pb.RegisterRecommendationServiceServer(srv, rec)
	pb.RegisterProductCatalogServiceServer(srv, fakeProductCatalogService{})
	go srv.Serve(lis)
	t.Cleanup(srv.Stop)

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return lis.DialContext(ctx) }),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("grpc.NewClient: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

func TestGetRecommendations_Disabled_NoRPC(t *testing.T) {
	rec := &fakeRecommendationService{ids: []string{"A", "B"}}
	conn := newBufconnServer(t, rec)
	fe := &frontendServer{
		recommendationsEnabled: false,
		recommendationSvcConn:  conn,
		productCatalogSvcConn:  conn,
	}

	got, err := fe.getRecommendations(context.Background(), "user-1", []string{"X"})
	if err != nil || got != nil {
		t.Fatalf("getRecommendations() = (%v, %v), want (nil, nil)", got, err)
	}
	if n := rec.callCount(); n != 0 {
		t.Errorf("ListRecommendations called %d times, want 0", n)
	}
}

func TestGetRecommendations_Enabled_CallsServiceAndCapsAt4(t *testing.T) {
	rec := &fakeRecommendationService{ids: []string{"A", "B", "C", "D", "E"}}
	conn := newBufconnServer(t, rec)
	fe := &frontendServer{
		recommendationsEnabled: true,
		recommendationSvcConn:  conn,
		productCatalogSvcConn:  conn,
	}

	got, err := fe.getRecommendations(context.Background(), "user-1", []string{"X", "Y"})
	if err != nil {
		t.Fatalf("getRecommendations() error: %v", err)
	}
	if len(got) != 4 {
		t.Errorf("got %d products, want 4", len(got))
	}
	if n := rec.callCount(); n != 1 {
		t.Fatalf("ListRecommendations called %d times, want 1", n)
	}
	req := rec.calls[0]
	if req.GetUserId() != "user-1" || strings.Join(req.GetProductIds(), ",") != "X,Y" {
		t.Errorf("request = {user_id: %q, product_ids: %v}, want {user-1, [X Y]}", req.GetUserId(), req.GetProductIds())
	}
}

// TestRecommendationsSectionHiddenWhenNil guards the {{ if $.recommendations }}
// check the page templates rely on to drop the strip when the flag is off.
func TestRecommendationsSectionHiddenWhenNil(t *testing.T) {
	tmpl, err := templates.Clone()
	if err != nil {
		t.Fatalf("clone templates: %v", err)
	}
	tmpl, err = tmpl.New("guard").Parse(`{{ if $.recommendations }}{{ template "recommendations" $ }}{{ end }}`)
	if err != nil {
		t.Fatalf("parse guard: %v", err)
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "guard", map[string]interface{}{"recommendations": []*pb.Product(nil)}); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if strings.Contains(buf.String(), `class="recommendations"`) {
		t.Errorf("recommendations section rendered for nil recommendations:\n%s", buf.String())
	}
}
