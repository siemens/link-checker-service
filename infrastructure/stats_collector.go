package infrastructure

import "sync"

// Stats of the link checker service
type Stats struct {
	IncomingRequests       int64
	OutgoingRequests       int64
	IncomingStreamRequests int64
	DnsResolutionsFailed   int64
	LinkChecksErrored      int64
	LinkChecksOk           int64
	LinkChecksBroken       int64
	LinkChecksDropped      int64
	LinkChecksSkipped      int64
}

type statsState struct {
	sync.RWMutex
	s Stats
}

var globalStatsState = newStatsState()

// GlobalStats returns the global handler to the stats collector
func GlobalStats() *statsState {
	return globalStatsState
}

// ResetGlobalStats the global stats
func ResetGlobalStats() {
	globalStatsState = newStatsState()
}

// OnIncomingRequest call on incoming request
func (stats *statsState) OnIncomingRequest() {
	stats.Lock()
	stats.s.IncomingRequests++
	stats.Unlock()
}

// OnIncomingStreamRequest called on an incoming stream request
func (stats *statsState) OnIncomingStreamRequest() {
	stats.Lock()
	stats.s.IncomingStreamRequests++
	stats.Unlock()
}

// OnOutgoingRequest called on outgoing request
func (stats *statsState) OnOutgoingRequest() {
	stats.Lock()
	stats.s.OutgoingRequests++
	stats.Unlock()
}

// OnDnsResolutionFailed called on dns resolution failure
func (stats *statsState) OnDnsResolutionFailed() {
	stats.Lock()
	stats.s.DnsResolutionsFailed++
	stats.Unlock()
}

// OnLinkErrored called on link check error
func (stats *statsState) OnLinkErrored() {
	stats.Lock()
	stats.s.LinkChecksErrored++
	stats.Unlock()
}

// OnLinkOk called on link check ok
func (stats *statsState) OnLinkOk() {
	stats.Lock()
	stats.s.LinkChecksOk++
	stats.Unlock()
}

// OnLinkBroken called on link check broken
func (stats *statsState) OnLinkBroken() {
	stats.Lock()
	stats.s.LinkChecksBroken++
	stats.Unlock()
}

// OnLinkDropped called on link check dropped
func (stats *statsState) OnLinkDropped() {
	stats.Lock()
	stats.s.LinkChecksDropped++
	stats.Unlock()
}

// OnLinkSkipped called on link check skipped
func (stats *statsState) OnLinkSkipped() {
	stats.Lock()
	stats.s.LinkChecksSkipped++
	stats.Unlock()
}

// GetStats returns a copy of the stats
func (stats *statsState) GetStats() Stats {
	stats.RLock()
	defer stats.RUnlock()
	return stats.s // a copy
}

func newStatsState() *statsState {
	return &statsState{}
}
