// Copyright 2020 Siemens AG
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
// SPDX-License-Identifier: MPL-2.0
package server

// URLRequest is a JSON structure for a single URL check request
type URLRequest struct {
	Context string `json:"context"` // will simply be echoed back for simpler reference of results
	URL     string `json:"url"`
}

// CheckURLsRequest is a JSON structure for a bulk URL check request
type CheckURLsRequest struct {
	Urls []URLRequest `json:"urls"`
}

// URLStatusResponse is the JSON response structure for one URL
type URLStatusResponse struct {
	// URLRequest is echoed back for correlation purposes
	URLRequest
	// Status is a stringified version of URLCheckStatus
	Status string `json:"status"`
	// HTTPStatus is the HTTP status received by the HTTP client (if available)
	HTTPStatus int `json:"http_status"`
	// Error contains the serialized error. Empty if no error was present
	Error string `json:"error"`
	// FetchedAtEpochSeconds indicates the UNIX timestamp in seconds at which the check has been performed
	FetchedAtEpochSeconds int64 `json:"timestamp"`
	// BodyPatternsFound will be filled with the configured regex patterns found in the response body
	BodyPatternsFound []string `json:"body_patterns_found"`
	// RemoteAddr is filled with the resolved address when `enableRequestTracing` is configured
	RemoteAddr string `json:"remote_addr,omitempty"`
}

// CheckURLsResponse is a JSON structure for the bulk URL check response
type CheckURLsResponse struct {
	Urls   []URLStatusResponse `json:"urls"`
	Result string              `json:"result"`
}
