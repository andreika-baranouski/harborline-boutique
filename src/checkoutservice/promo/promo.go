// Package promo holds the configured promo code table and decides whether a
// code applies to an order placed at a given time.
package promo

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	pb "github.com/GoogleCloudPlatform/microservices-demo/src/checkoutservice/genproto"
)

const (
	// MaxCodeLength is the longest code accepted; longer input is UNKNOWN.
	MaxCodeLength = 32
	dateLayout    = "2006-01-02"
)

// Code is one entry of the promo table. Start and End are inclusive UTC days;
// nil means the window is open on that side.
type Code struct {
	Code    string
	Percent uint32
	Start   *time.Time
	End     *time.Time
}

// Table maps a normalised code to its entry.
type Table map[string]Code

type rawCode struct {
	Code    string `json:"code"`
	Percent uint32 `json:"percent"`
	Start   string `json:"start"`
	End     string `json:"end"`
}

// Normalize trims the code and upper-cases it.
func Normalize(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

// validFormat reports whether a normalised code is 1..MaxCodeLength ASCII
// letters and digits.
func validFormat(code string) bool {
	if code == "" || len(code) > MaxCodeLength {
		return false
	}
	for _, c := range code {
		if !(c >= 'A' && c <= 'Z' || c >= '0' && c <= '9') {
			return false
		}
	}
	return true
}

// Parse reads the promo table from its JSON form, e.g.
//
//	[{"code":"WELCOME10","percent":10},
//	 {"code":"HARBOR25","percent":25,"end":"2026-12-31"}]
//
// An empty string yields an empty table. Invalid JSON yields an empty table and
// one error. An invalid entry is skipped with an error; the others are kept.
func Parse(raw string) (Table, []error) {
	t := Table{}
	if strings.TrimSpace(raw) == "" {
		return t, nil
	}
	var entries []rawCode
	if err := json.Unmarshal([]byte(raw), &entries); err != nil {
		return t, []error{fmt.Errorf("promo codes: invalid JSON: %w", err)}
	}
	var errs []error
	for i, e := range entries {
		c, err := parseEntry(e)
		if err == nil {
			if _, dup := t[c.Code]; dup {
				err = fmt.Errorf("duplicate code %q", c.Code)
			}
		}
		if err != nil {
			errs = append(errs, fmt.Errorf("promo codes: entry #%d skipped: %w", i, err))
			continue
		}
		t[c.Code] = c
	}
	return t, errs
}

func parseEntry(e rawCode) (Code, error) {
	c := Code{Code: Normalize(e.Code), Percent: e.Percent}
	if !validFormat(c.Code) {
		return c, fmt.Errorf("code %q must be 1-%d letters or digits", e.Code, MaxCodeLength)
	}
	if c.Percent < 1 || c.Percent > 100 {
		return c, fmt.Errorf("code %q: percent %d outside 1-100", c.Code, c.Percent)
	}
	var err error
	if c.Start, err = parseDate(e.Start); err != nil {
		return c, fmt.Errorf("code %q: start: %w", c.Code, err)
	}
	if c.End, err = parseDate(e.End); err != nil {
		return c, fmt.Errorf("code %q: end: %w", c.Code, err)
	}
	if c.Start != nil && c.End != nil && c.Start.After(*c.End) {
		return c, fmt.Errorf("code %q: start %s is after end %s", c.Code, e.Start, e.End)
	}
	return c, nil
}

func parseDate(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	d, err := time.ParseInLocation(dateLayout, s, time.UTC)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// Codes returns the codes in the table, for logging.
func (t Table) Codes() []string {
	out := make([]string, 0, len(t))
	for k := range t {
		out = append(out, k)
	}
	return out
}

// Evaluate decides the outcome of code at time now. The result carries the
// normalised code, the status and, when APPLIED, the percentage; the discount
// amount is left for the caller to fill in.
func (t Table) Evaluate(code string, now time.Time) *pb.PromoResult {
	n := Normalize(code)
	res := &pb.PromoResult{Code: n, Status: pb.PromoResult_UNKNOWN}
	if !validFormat(n) {
		return res
	}
	c, ok := t[n]
	if !ok {
		return res
	}
	now = now.UTC()
	if c.Start != nil && now.Before(*c.Start) {
		res.Status = pb.PromoResult_NOT_YET_ACTIVE
		return res
	}
	if c.End != nil && !now.Before(c.End.AddDate(0, 0, 1)) {
		res.Status = pb.PromoResult_EXPIRED
		return res
	}
	res.Status = pb.PromoResult_APPLIED
	res.Percent = c.Percent
	return res
}
