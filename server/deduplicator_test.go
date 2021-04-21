// Copyright 2020-2021 Siemens AG
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
// SPDX-License-Identifier: MPL-2.0
package server

import (
	"testing"

	"github.com/siemens/link-checker-service/infrastructure"
	"github.com/stretchr/testify/assert"
)

func TestDeduplicator(t *testing.T) {
	request := []URLRequest{
		{
			Context: "0",
			URL:     "http://a",
		},
		{
			Context: "1",
			URL:     "b",
		},
		{
			Context: "2",
			URL:     "http://a ",
		},
		{
			Context: "3",
			URL:     "b",
		},
		{
			Context: "4",
			URL:     " http://a",
		},
	}
	urls := deduplicateURLs(request)

	// checking internals for sanity
	assert.Len(t, urls.toCheck, 2)
	assert.Len(t, urls.toDuplicate, 2)             // a and b
	assert.Len(t, urls.toDuplicate["http://a"], 2) // all duplicate a

	aResponse := URLStatusResponse{
		URLRequest:            request[0],
		Status:                "ok",
		HTTPStatus:            infrastructure.CustomHTTPErrorCode,
		Error:                 "error!",
		FetchedAtEpochSeconds: 0,
		BodyPatternsFound:     []string{"a"},
	}

	// simulate an existing result for a
	urls.onResponse(&aResponse)

	a := urls.allResultsDeduplicated([]URLStatusResponse{aResponse})
	assert.Len(t, a, 3, "should have seen a response, thus have deduplicated")
	aResponse2 := aResponse
	aResponse2.URLRequest = request[2]
	assert.Equal(t, a[0], aResponse, "duplicated result should have contained the original response first")
	assert.Equal(t, a[1], aResponse2, "duplicated result should have contained a response with the original url & context")
	a = urls.deduplicatedResultFor(aResponse)
	assert.Len(t, a, 3, "should have returned a duplicated a result")

	bResponse := aResponse
	bResponse.URLRequest = request[1]

	b := urls.allResultsDeduplicated([]URLStatusResponse{aResponse})
	assert.Len(t, b, 3, "should not have seen the b response, thus, still returning a results")

	urls.onResponse(&bResponse)

	all := urls.allResultsDeduplicated([]URLStatusResponse{aResponse, bResponse})
	assert.Len(t, all, 5, "all request urls should be present in the response")

	onlyB := urls.deduplicatedResultFor(bResponse)
	assert.Len(t, onlyB, 2, "should only contain b results")
}
