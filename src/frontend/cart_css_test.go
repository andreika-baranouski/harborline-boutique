package main

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// TestCartCSSNoWrapRulesOnlyAboveTablet guards against the cart page
// overflowing horizontally on phones: the rules that stop the cart summary
// from wrapping must only apply from the tablet breakpoint (768px) up.
func TestCartCSSNoWrapRulesOnlyAboveTablet(t *testing.T) {
	b, err := os.ReadFile("static/styles/cart.css")
	if err != nil {
		t.Fatalf("read cart.css: %v", err)
	}
	css := regexp.MustCompile(`(?s)/\*.*?\*/`).ReplaceAllString(string(b), "")

	tabletMedia := regexp.MustCompile(`^@media\s*\(\s*min-width:\s*768px\s*\)$`)
	guarded := []string{"min-width: 560px", "flex-wrap: nowrap"}

	// Walk the stylesheet tracking the enclosing at-rule of each declaration.
	var stack []string
	start := 0
	for i, c := range css {
		switch c {
		case '{':
			stack = append(stack, strings.TrimSpace(css[start:i]))
			start = i + 1
		case '}':
			body := css[start:i]
			for _, decl := range guarded {
				if !strings.Contains(body, decl) {
					continue
				}
				if len(stack) < 2 || !tabletMedia.MatchString(stack[len(stack)-2]) {
					t.Errorf("%q in %q must be inside @media (min-width: 768px)", decl, stack[len(stack)-1])
				}
			}
			stack = stack[:len(stack)-1]
			start = i + 1
		}
	}
}
