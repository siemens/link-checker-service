// Copyright 2020-2024 Siemens AG
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
// SPDX-License-Identifier: MPL-2.0

package infrastructure

import (
	"maps"
	"sync"
)

const (
	erroredStatus             = "errored"
	dnsResolutionFailedStatus = "dns_resolution_failed"
	droppedStatus             = "dropped"
	skippedStatus             = "skipped"
)

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
}

// DomainStatsResponse for all domains
type DomainStatsResponse struct {
	DomainStats map[string]DomainStats
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
	d map[string]DomainStats
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
	stats.incrementOrDefaultStatus(domain, dnsResolutionFailedStatus)
	stats.Unlock()
}

// OnLinkErrored called on link check error
func (stats *StatsState) OnLinkErrored(domain string) {
	stats.Lock()
	stats.s.LinkChecksErrored++
	stats.incrementOrDefaultStatus(domain, erroredStatus)
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
	stats.incrementOrDefaultStatus(domain, droppedStatus)
	stats.Unlock()
}

// OnLinkSkipped called on link check skipped
func (stats *StatsState) OnLinkSkipped(domain string) {
	stats.Lock()
	stats.s.LinkChecksSkipped++
	stats.incrementOrDefaultStatus(domain, skippedStatus)
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

// GetDomainStats returns a copy of the detailed domain stats
func (stats *StatsState) GetDomainStats() DomainStatsResponse {
	stats.RLock()
	defer stats.RUnlock()
	deepClone := make(map[string]DomainStats)
	for k, v := range stats.d {
		deepClone[k] = DomainStats{
			Ok:            v.Ok,
			BrokenBecause: maps.Clone(v.BrokenBecause),
		}
	}
	return DomainStatsResponse{deepClone} // a copy
}

func (stats *StatsState) incrementOrDefaultOk(domain string) {
	if _, ok := stats.d[domain]; !ok {
		stats.d[domain] = defaultDomainStats()
	}
	ds := stats.d[domain]
	ds.Ok++
	stats.d[domain] = ds
}

func (stats *StatsState) incrementOrDefaultStatus(domain string, status string) {
	if _, ok := stats.d[domain]; !ok {
		stats.d[domain] = defaultDomainStats()
	}
	ds := stats.d[domain]
	ds.BrokenBecause = incrementOrDefaultBrokenBecause(ds.BrokenBecause, status)
	stats.d[domain] = ds
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
		s: Stats{},
		d: map[string]DomainStats{},
	}
}
