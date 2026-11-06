package promo

import (
	"strings"
	"testing"
	"time"

	pb "github.com/GoogleCloudPlatform/microservices-demo/src/checkoutservice/genproto"
)

const launchConfig = `[{"code":"WELCOME10","percent":10},
 {"code":"HARBOR25","percent":25,"end":"2026-12-31"}]`

func at(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestParseLaunchConfig(t *testing.T) {
	tbl, errs := Parse(launchConfig)
	if len(errs) != 0 {
		t.Fatalf("Parse() errors = %v", errs)
	}
	if len(tbl) != 2 {
		t.Fatalf("Parse() loaded %d codes, want 2", len(tbl))
	}
	if c := tbl["WELCOME10"]; c.Percent != 10 || c.Start != nil || c.End != nil {
		t.Errorf("WELCOME10 = %+v, want 10%% open-ended", c)
	}
	if c := tbl["HARBOR25"]; c.Percent != 25 || c.End == nil || c.End.Format(dateLayout) != "2026-12-31" {
		t.Errorf("HARBOR25 = %+v, want 25%% until 2026-12-31", c)
	}
}

func TestParseEmpty(t *testing.T) {
	for _, raw := range []string{"", "  ", "[]"} {
		tbl, errs := Parse(raw)
		if len(tbl) != 0 || len(errs) != 0 {
			t.Errorf("Parse(%q) = %v, %v; want empty table, no errors", raw, tbl, errs)
		}
	}
}

func TestParseInvalidJSON(t *testing.T) {
	tbl, errs := Parse(`{"code":`)
	if len(tbl) != 0 {
		t.Errorf("Parse(invalid) loaded %v, want empty table", tbl)
	}
	if len(errs) != 1 {
		t.Errorf("Parse(invalid) errors = %v, want 1", errs)
	}
}

func TestParseSkipsBadEntries(t *testing.T) {
	raw := `[
	 {"code":"GOOD5","percent":5},
	 {"code":"ZERO","percent":0},
	 {"code":"TOOMUCH","percent":101},
	 {"code":"BADDATE","percent":10,"end":"2026-13-01"},
	 {"code":"BACKWARDS","percent":10,"start":"2026-12-31","end":"2026-12-01"},
	 {"code":"good5","percent":50},
	 {"code":"BAD CODE","percent":10},
	 {"code":"` + strings.Repeat("A", MaxCodeLength+1) + `","percent":10}
	]`
	tbl, errs := Parse(raw)
	if len(errs) != 7 {
		t.Errorf("Parse() errors = %d (%v), want 7", len(errs), errs)
	}
	if len(tbl) != 1 || tbl["GOOD5"].Percent != 5 {
		t.Errorf("Parse() table = %v, want only GOOD5 at 5%%", tbl)
	}
}

func TestEvaluate(t *testing.T) {
	tbl, _ := Parse(launchConfig)
	tbl2, _ := Parse(`[{"code":"WINTER","percent":15,"start":"2026-12-01","end":"2026-12-31"}]`)
	for k, v := range tbl2 {
		tbl[k] = v
	}
	tests := []struct {
		name        string
		code        string
		now         string
		wantCode    string
		wantStatus  pb.PromoResult_Status
		wantPercent uint32
	}{
		{"unknown", "NOPE", "2026-11-06T12:00:00Z", "NOPE", pb.PromoResult_UNKNOWN, 0},
		{"lower case and spaces", "  welcome10 ", "2026-11-06T12:00:00Z", "WELCOME10", pb.PromoResult_APPLIED, 10},
		{"open-ended far future", "WELCOME10", "2099-01-01T00:00:00Z", "WELCOME10", pb.PromoResult_APPLIED, 10},
		{"last second of end day", "HARBOR25", "2026-12-31T23:59:59Z", "HARBOR25", pb.PromoResult_APPLIED, 25},
		{"day after end", "HARBOR25", "2027-01-01T00:00:00Z", "HARBOR25", pb.PromoResult_EXPIRED, 0},
		{"end checked in UTC", "HARBOR25", "2026-12-31T20:00:00-05:00", "HARBOR25", pb.PromoResult_EXPIRED, 0},
		{"day before start", "WINTER", "2026-11-30T23:59:59Z", "WINTER", pb.PromoResult_NOT_YET_ACTIVE, 0},
		{"first second of start day", "WINTER", "2026-12-01T00:00:00Z", "WINTER", pb.PromoResult_APPLIED, 15},
		{"over-long code", strings.Repeat("A", MaxCodeLength+1), "2026-11-06T12:00:00Z", strings.Repeat("A", MaxCodeLength+1), pb.PromoResult_UNKNOWN, 0},
		{"malformed code", "WEL-COME10", "2026-11-06T12:00:00Z", "WEL-COME10", pb.PromoResult_UNKNOWN, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tbl.Evaluate(tt.code, at(tt.now))
			if got.GetCode() != tt.wantCode || got.GetStatus() != tt.wantStatus || got.GetPercent() != tt.wantPercent {
				t.Errorf("Evaluate(%q, %s) = {%q %v %d}, want {%q %v %d}", tt.code, tt.now,
					got.GetCode(), got.GetStatus(), got.GetPercent(), tt.wantCode, tt.wantStatus, tt.wantPercent)
			}
			if got.GetDiscount() != nil {
				t.Errorf("Evaluate() set discount %v, want nil", got.GetDiscount())
			}
		})
	}
}

func TestEvaluateEmptyTable(t *testing.T) {
	if got := (Table{}).Evaluate("WELCOME10", time.Now()); got.GetStatus() != pb.PromoResult_UNKNOWN {
		t.Errorf("empty table Evaluate() status = %v, want UNKNOWN", got.GetStatus())
	}
}
