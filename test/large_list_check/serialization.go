// Copyright 2020-2022 Siemens AG
// This Source Code Form is subject to the terms of
// Attribution-ShareAlike 4.0 International (CC BY-SA 4.0) license
// https://creativecommons.org/licenses/by-sa/4.0/
// SPDX-License-Identifier: CC-BY-SA-4.0
package main

// copied from .../link-checker-service/server to decouple the sample project

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
}

// CheckURLsResponse is a JSON structure for the bulk URL check response
type CheckURLsResponse struct {
	Urls   []URLStatusResponse `json:"urls"`
	Result string              `json:"result"`
}
