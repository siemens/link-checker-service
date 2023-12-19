// Copyright 2020-2023 Siemens AG
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
// SPDX-License-Identifier: MPL-2.0

package infrastructure

import (
	"encoding/json"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCollectingFromSeveralGoroutines(t *testing.T) {
	ResetGlobalStats()
	const goroutines = 32
	const requests = 100000
	addStats(goroutines, requests)
	expectedCount := int64(goroutines * requests)
	s := GlobalStats().GetStats()
	d := GlobalStats().GetDomainStats()
	assert.Equal(t, Stats{
		IncomingRequests:       expectedCount,
		OutgoingRequests:       expectedCount,
		IncomingStreamRequests: expectedCount,
		DNSResolutionsFailed:   expectedCount,
		LinkChecksErrored:      expectedCount,
		LinkChecksOk:           expectedCount,
		LinkChecksBroken:       expectedCount,
		LinkChecksDropped:      expectedCount,
		LinkChecksSkipped:      expectedCount,
		CacheHits:              expectedCount,
		CacheMisses:            expectedCount,
	}, s)

	assert.Equal(t, map[string]DomainStats{
		"example.com": {
			BrokenBecause: map[string]int64{}, // not nil!
			Ok:            expectedCount,
		},
		"notfound.com": {
			BrokenBecause: map[string]int64{
				"404": expectedCount,
			},
		},
		"bad-domain.com": {
			BrokenBecause: map[string]int64{
				"dns_resolution_failed": expectedCount,
			},
		},
		"dropped.com": {
			BrokenBecause: map[string]int64{
				"dropped": expectedCount,
			},
		},
		"errored.com": {
			BrokenBecause: map[string]int64{
				"errored": expectedCount,
			},
		},
		"skipped.com": {
			BrokenBecause: map[string]int64{
				"skipped": expectedCount,
			},
		},
	}, d.DomainStats)
}

func TestDomainsStatsSerialization(t *testing.T) {
	stats := DomainStatsResponse{
		DomainStats: map[string]DomainStats{
			"example.com": {
				BrokenBecause: map[string]int64{
					"200":     33,
					"dropped": 22,
				},
				Ok: 11,
			},
		},
	}

	// serialize
	bytes, err := json.Marshal(&stats)
	assert.NoError(t, err)

	// deserialize
	deserialized := DomainStatsResponse{}
	err = json.Unmarshal(bytes, &deserialized)
	assert.NoError(t, err)

	// check round-trip
	assert.Equal(t, deserialized.DomainStats["example.com"].BrokenBecause["200"], int64(33))
	assert.Equal(t, deserialized.DomainStats["example.com"].BrokenBecause["dropped"], int64(22))
	assert.Equal(t, deserialized.DomainStats["example.com"].Ok, int64(11))
}

func addStats(numGoroutines int, count int) {
	var wg sync.WaitGroup

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func() {
			for n := 0; n < count; n++ {
				s := GlobalStats()
				s.OnIncomingRequest()
				s.OnOutgoingRequest()
				s.OnIncomingStreamRequest()
				s.OnLinkBroken("notfound.com", "404")
				s.OnLinkDropped("dropped.com")
				s.OnDNSResolutionFailed("bad-domain.com")
				s.OnLinkErrored("errored.com")
				s.OnLinkOk("example.com")
				s.OnLinkSkipped("skipped.com")
				s.OnCacheHit()
				s.OnCacheMiss()
			}
			defer wg.Done()
		}()

	}
	wg.Wait()
}
