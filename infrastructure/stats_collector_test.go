package infrastructure

import (
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
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
		DnsResolutionsFailed:   expectedCount,
		LinkChecksErrored:      expectedCount,
		LinkChecksOk:           expectedCount,
		LinkChecksBroken:       expectedCount,
		LinkChecksDropped:      expectedCount,
		LinkChecksSkipped:      expectedCount,
	}, s)

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
				s.OnLinkBroken()
				s.OnLinkDropped()
				s.OnDnsResolutionFailed()
				s.OnLinkErrored()
				s.OnLinkOk()
				s.OnLinkSkipped()
			}
			defer wg.Done()
		}()

	}
	wg.Wait()
}
