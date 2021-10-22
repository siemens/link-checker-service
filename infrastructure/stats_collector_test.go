package infrastructure

import (
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
)

func TestCollectingFromManyGoroutines(t *testing.T) {
	ResetGlobalStats()
	const goroutines = 32
	const requests = 100000
	addStats(goroutines, requests)
	assert.Equal(t, goroutines*requests, GlobalStats().GetStats().requests)
}

func addStats(numGoroutines int, count int) {
	var wg sync.WaitGroup

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func() {
			for n := 0; n < count; n++ {
				GlobalStats().OnRequest()
			}
			defer wg.Done()
		}()

	}
	wg.Wait()
}
