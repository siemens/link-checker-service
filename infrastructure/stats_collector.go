// Copyright 2020-2022 Siemens AG
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
// SPDX-License-Identifier: MPL-2.0

package infrastructure

import "sync"

// Stats of the link checker service
type Stats struct {
	IncomingRequests       int64
	OutgoingRequests       int64
	IncomingStreamRequests int64
	DNSResolutionsFailed   int64
	LinkChecksErrored      int64
	LinkChecksOk           int64
	LinkChecksBroken       int64
	LinkChecksDropped      int64
	LinkChecksSkipped      int64
	CacheHits              int64
	CacheMisses            int64
	DomainStats            map[string]DomainStats
}

// DomainStats for one domain
type DomainStats struct {
	BrokenBecause map[string]int64
	Ok            int64
}

// StatsState is the protected instance of the Stats object
type StatsState struct {
	sync.RWMutex
	s Stats
}

var globalStatsState = newStatsState()

// GlobalStats returns the global handler to the stats collector
func GlobalStats() *StatsState {
	return globalStatsState
}

// ResetGlobalStats the global stats
func ResetGlobalStats() {
	globalStatsState = newStatsState()
}

// OnIncomingRequest call on incoming request
func (stats *StatsState) OnIncomingRequest() {
	stats.Lock()
	stats.s.IncomingRequests++
	stats.Unlock()
}

// OnIncomingStreamRequest called on an incoming stream request
func (stats *StatsState) OnIncomingStreamRequest() {
	stats.Lock()
	stats.s.IncomingStreamRequests++
	stats.Unlock()
}

// OnOutgoingRequest called on outgoing request
func (stats *StatsState) OnOutgoingRequest() {
	stats.Lock()
	stats.s.OutgoingRequests++
	stats.Unlock()
}

// OnDNSResolutionFailed called on dns resolution failure
func (stats *StatsState) OnDNSResolutionFailed(domain string) {
	stats.Lock()
	stats.s.DNSResolutionsFailed++
	stats.incrementOrDefaultStatus(domain, "dns_resolution_failed")
	stats.Unlock()
}

// OnLinkErrored called on link check error
func (stats *StatsState) OnLinkErrored(domain string) {
	stats.Lock()
	stats.s.LinkChecksErrored++
	stats.incrementOrDefaultStatus(domain, "errored")
	stats.Unlock()
}

// OnLinkOk called on link check ok
func (stats *StatsState) OnLinkOk(domain string) {
	stats.Lock()
	stats.s.LinkChecksOk++
	stats.incrementOrDefaultOk(domain)
	stats.Unlock()
}

func defaultDomainStats() DomainStats {
	return DomainStats{
		BrokenBecause: map[string]int64{},
	}
}

// OnLinkBroken called on link check broken
func (stats *StatsState) OnLinkBroken(domain string, status string) {
	stats.Lock()
	stats.s.LinkChecksBroken++
	stats.incrementOrDefaultStatus(domain, status)
	stats.Unlock()
}

// OnLinkDropped called on link check dropped
func (stats *StatsState) OnLinkDropped(domain string) {
	stats.Lock()
	stats.s.LinkChecksDropped++
	stats.incrementOrDefaultStatus(domain, "dropped")
	stats.Unlock()
}

// OnLinkSkipped called on link check skipped
func (stats *StatsState) OnLinkSkipped(domain string) {
	stats.Lock()
	stats.s.LinkChecksSkipped++
	stats.incrementOrDefaultStatus(domain, "skipped")
	stats.Unlock()
}

// OnCacheHit called when the result is taken from the cache
func (stats *StatsState) OnCacheHit() {
	stats.Lock()
	stats.s.CacheHits++
	stats.Unlock()
}

// OnCacheMiss called when the requested URL wasn't found in the cache
func (stats *StatsState) OnCacheMiss() {
	stats.Lock()
	stats.s.CacheMisses++
	stats.Unlock()
}

// GetStats returns a copy of the stats
func (stats *StatsState) GetStats() Stats {
	stats.RLock()
	defer stats.RUnlock()
	return stats.s // a copy
}

func (stats *StatsState) incrementOrDefaultOk(domain string) {
	if _, ok := stats.s.DomainStats[domain]; !ok {
		stats.s.DomainStats[domain] = defaultDomainStats()
	}
	ds := stats.s.DomainStats[domain]
	ds.Ok++
	stats.s.DomainStats[domain] = ds
}

func (stats *StatsState) incrementOrDefaultStatus(domain string, status string) {
	if _, ok := stats.s.DomainStats[domain]; !ok {
		stats.s.DomainStats[domain] = defaultDomainStats()
	}
	ds := stats.s.DomainStats[domain]
	ds.BrokenBecause = incrementOrDefaultBrokenBecause(ds.BrokenBecause, status)
	stats.s.DomainStats[domain] = ds
}

func incrementOrDefaultBrokenBecause(because map[string]int64, status string) map[string]int64 {
	count := int64(0)

	if c, ok := because[status]; ok {
		count = c
	}

	because[status] = count + 1

	return because
}

func newStatsState() *StatsState {
	return &StatsState{
		s: Stats{
			DomainStats: map[string]DomainStats{},
		},
	}
}
