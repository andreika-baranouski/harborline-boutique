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

// Package delivery formats estimated delivery dates for display.
package delivery

import "time"

// isoLayout is the wire format of estimated_delivery: an ISO-8601 calendar date.
const isoLayout = "2006-01-02"

// Format renders an ISO "YYYY-MM-DD" date in the en-US long format
// ("Thursday, November 12, 2026"), matching the confirmation e-mail.
// It returns "" for an empty or unparseable value.
func Format(iso string) string {
	if iso == "" {
		return ""
	}
	d, err := time.Parse(isoLayout, iso)
	if err != nil {
		return ""
	}
	return d.Format("Monday, January 2, 2006")
}
