// Copyright 2020-2023 Siemens AG
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
// SPDX-License-Identifier: MPL-2.0

package server

import "sync"

type deduplicator struct {
	toCheck       []URLRequest
	toDuplicate   map[string][]URLRequest // multimap
	responseCache sync.Map                // stores *URLStatusResponse instances
}

func deduplicateURLs(urls []URLRequest) *deduplicator {
	res := &deduplicator{
		toCheck:       []URLRequest{},
		toDuplicate:   map[string][]URLRequest{},
		responseCache: sync.Map{},
	}

	seen := map[string]struct{}{}

	for _, u := range urls {
		key := normalizedURL(u.URL)
		if _, ok := seen[key]; ok {
			// if seen -> duplicate
			if s, ok := res.toDuplicate[key]; ok {
				s = append(s, u)
				res.toDuplicate[key] = s
			} else {
				res.toDuplicate[key] = []URLRequest{u}
			}
		} else {
			// otherwise -> check
			seen[key] = struct{}{}
			res.toCheck = append(res.toCheck, u)
		}
	}

	return res
}

func (urls *deduplicator) deduplicatedResultFor(result URLStatusResponse) []URLStatusResponse {
	res := []URLStatusResponse{result}

	if requestSet, ok := urls.toDuplicate[normalizedURL(result.URL)]; ok {
		for _, u := range requestSet {
			res = urls.addResponseIfCached(u, res)
		}
	}

	return res
}

func (urls *deduplicator) allResultsDeduplicated(results []URLStatusResponse) []URLStatusResponse {
	res := results
	for _, requestSet := range urls.toDuplicate {
		for _, u := range requestSet {
			res = urls.addResponseIfCached(u, res)
		}
	}
	return res
}

func (urls *deduplicator) addResponseIfCached(u URLRequest, res []URLStatusResponse) []URLStatusResponse {
	key := normalizedURL(u.URL)

	if cached, ok := urls.responseCache.Load(key); ok && cached != nil {
		response := cached.(*URLStatusResponse)
		// copy
		var newResponse = *response
		// replace the request context & url to the original of the request
		newResponse.Context = u.Context
		newResponse.URL = u.URL
		res = append(res, newResponse)
	}
	return res
}

func (urls *deduplicator) onResponse(response *URLStatusResponse) {
	urls.responseCache.Store(normalizedURL(response.URL), response)
}
