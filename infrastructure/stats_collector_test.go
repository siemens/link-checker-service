// Copyright 2020-2022 Siemens AG
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
		DomainStats: map[string]DomainStats{
			"example.com": {
				BrokenBecause: map[string]int64{}, // not nil!
				Ok:            expectedCount,
			},
			"notfound.com": {
				BrokenBecause: map[string]int64{
					"404": expectedCount,
				},
				Ok: 0,
			},
		},
	}, s)

}

func TestSerialization(t *testing.T) {
	stats := Stats{
		IncomingRequests: 42,
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
	deserialized := Stats{}
	err = json.Unmarshal(bytes, &deserialized)
	assert.NoError(t, err)

	// check round-trip
	assert.Equal(t, deserialized.IncomingRequests, int64(42))
	assert.Equal(t, deserialized.LinkChecksOk, int64(0))

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
				s.OnLinkDropped()
				s.OnDNSResolutionFailed()
				s.OnLinkErrored()
				s.OnLinkOk("example.com")
				s.OnLinkSkipped()
				s.OnCacheHit()
				s.OnCacheMiss()
			}
			defer wg.Done()
		}()

	}
	wg.Wait()
}
