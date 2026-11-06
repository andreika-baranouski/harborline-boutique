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
	"time"
)

// deliveryDateLayout is the wire format of estimated_delivery: an ISO-8601 calendar date.
const deliveryDateLayout = "2006-01-02"

// defaultTransitDays is the business-day transit time for destinations without an entry in transitDays.
const defaultTransitDays = 7

// transitDays overrides defaultTransitDays for specific destination countries,
// keyed by the normalized country (see normalizeCountry).
var transitDays = map[string]int{
	"US": 3,
}

// addBusinessDays returns the date n business days (Mon–Fri) after d; weekends are not counted,
// so a shipment leaving on a Saturday or Sunday starts counting on Monday.
func addBusinessDays(d time.Time, n int) time.Time {
	for n > 0 {
		d = d.AddDate(0, 0, 1)
		if wd := d.Weekday(); wd != time.Saturday && wd != time.Sunday {
			n--
		}
	}
	return d
}

// EstimateDelivery returns the estimated delivery date for a shipment leaving on shipDate to country.
// The ship date is taken as its UTC calendar date.
func EstimateDelivery(shipDate time.Time, country string) time.Time {
	days, ok := transitDays[normalizeCountry(country)]
	if !ok {
		days = defaultTransitDays
	}
	y, m, d := shipDate.UTC().Date()
	return addBusinessDays(time.Date(y, m, d, 0, 0, 0, 0, time.UTC), days)
}
