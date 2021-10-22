package infrastructure

import "sync"

type Stats struct {
	requests int
}

type statsState struct {
	sync.RWMutex
	s Stats
}

var globalStatsState = newStatsState()

func GlobalStats() *statsState {
	return globalStatsState
}

func ResetGlobalStats() {
	globalStatsState = newStatsState()
}

func newStatsState() *statsState {
	return &statsState{}
}

func (stats *statsState) OnRequest() {
	stats.Lock()
	stats.s.requests++
	stats.Unlock()
}

func (stats *statsState) GetStats() Stats {
	return stats.s
}
