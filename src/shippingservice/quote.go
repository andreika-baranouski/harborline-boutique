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
	"fmt"
	"math"
	"strings"
)

// Quote represents a currency value.
type Quote struct {
	Dollars uint32
	Cents   uint32
}

// String representation of the Quote.
func (q Quote) String() string {
	return fmt.Sprintf("$%d.%d", q.Dollars, q.Cents)
}

// defaultRate is the flat per-shipment rate for destinations without a regional rate.
const defaultRate = 8.99

// regionalRates overrides defaultRate for specific destination countries,
// keyed by the normalized country (see normalizeCountry).
var regionalRates = map[string]float64{
	"US": 8.99,
}

// countryAliases maps the ways customers type a country (upper-cased) to its regionalRates key.
var countryAliases = map[string]string{
	"USA":                      "US",
	"U.S.":                     "US",
	"UNITED STATES":            "US",
	"UNITED STATES OF AMERICA": "US",
}

// normalizeCountry trims and upper-cases a free-text country and resolves known aliases.
func normalizeCountry(country string) string {
	c := strings.ToUpper(strings.TrimSpace(country))
	if alias, ok := countryAliases[c]; ok {
		return alias
	}
	return c
}

// CreateQuoteFromCount takes a number of items and the destination country and returns a shipping quote.
func CreateQuoteFromCount(count int, country string) Quote {
	if count == 0 {
		return CreateQuoteFromFloat(0)
	}
	rate, ok := regionalRates[normalizeCountry(country)]
	if !ok {
		rate = defaultRate
	}
	return CreateQuoteFromFloat(rate)
}

// CreateQuoteFromFloat takes a price represented as a float and creates a Price struct.
func CreateQuoteFromFloat(value float64) Quote {
	units, fraction := math.Modf(value)
	return Quote{
		uint32(units),
		uint32(math.Trunc(fraction * 100)),
	}
}