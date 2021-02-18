// Copyright 2020 Siemens AG
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
// SPDX-License-Identifier: MPL-2.0
package infrastructure

import (
	"context"
	"fmt"
	"github.com/dgraph-io/ristretto"
	"log"
	"time"

	"github.com/spf13/viper"

	"github.com/patrickmn/go-cache"
)

const defaultCacheExpirationInterval = 24 * time.Hour
const defaultCacheCleanupInterval = 48 * time.Hour
const defaultRetryFailedAfter = 30 * time.Second

// CachedURLChecker wraps a concurrency-limited URL checker
type CachedURLChecker struct {
	cache                   *cache.Cache
	cacheSettings           cacheSettings
	ristrettoCache          *ristretto.Cache
	retryFailedAfterSeconds int64

	ccLimitedChecker *CCLimitedURLChecker
}

type cacheSettings struct {
	cacheExpirationInterval time.Duration
	cacheCleanupInterval    time.Duration
	retryFailedAfter        time.Duration
}

// NewCachedURLChecker creates a new cached URL checker instance
func NewCachedURLChecker() *CachedURLChecker {
	// https://github.com/dgraph-io/ristretto#Config
	ristrettoCache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e6,     // number of keys to track frequency of (~10x 100k links).
		MaxCost:     1 << 30, // maximum cost of cache (1GB).
		BufferItems: 64,      // number of keys per Get buffer: as recommended
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(ristrettoCache)

	settings := fetchCachedURLCheckerSettings()
	checker := CachedURLChecker{
		cache:                   cache.New(settings.cacheExpirationInterval, settings.cacheCleanupInterval),
		ristrettoCache:          ristrettoCache,
		cacheSettings:           settings,
		ccLimitedChecker:        NewCCLimitedURLChecker(),
		retryFailedAfterSeconds: int64(settings.retryFailedAfter.Seconds()),
	}
	return &checker
}

func fetchCachedURLCheckerSettings() cacheSettings {
	s := cacheSettings{}

	cacheExpirationInterval := viper.GetString("cacheExpirationInterval")
	if d, err := time.ParseDuration(cacheExpirationInterval); err != nil {
		log.Printf("Ignoring cacheExpirationInterval %v -> %v (%v)", cacheExpirationInterval, defaultCacheExpirationInterval, err)
	} else {
		s.cacheExpirationInterval = d
		log.Printf("cacheExpirationInterval: %v", cacheExpirationInterval)
	}

	cacheCleanupInterval := viper.GetString("cacheCleanupInterval")
	if d, err := time.ParseDuration(cacheCleanupInterval); err != nil {
		log.Printf("Ignoring cacheCleanupInterval %v -> %v (%v)", cacheCleanupInterval, defaultCacheCleanupInterval, err)
	} else {
		log.Printf("cacheCleanupInterval: %v", cacheCleanupInterval)
		s.cacheCleanupInterval = d
	}

	retryFailedAfter := viper.GetString("retryFailedAfter")
	if d, err := time.ParseDuration(retryFailedAfter); err != nil {
		log.Printf("Ignoring retryFailedAfter %v -> %v (%v)", cacheCleanupInterval, defaultRetryFailedAfter, err)
	} else {
		log.Printf("retryFailedAfter: %v", retryFailedAfter)
		s.retryFailedAfter = d
	}
	return s
}

// CheckURL checks the desired URL
func (c *CachedURLChecker) CheckURL(ctx context.Context, url string) *URLCheckResult {
	//value, found := c.cache.Get(url)
	value, found := c.ristrettoCache.Get(url)

	if found {
		res := value.(*URLCheckResult)

		// failures could have been temporary -> retry a URL after some time
		if c.shouldTakeCachedResult(res) {
			return res
		}
	}

	// otherwise, do the check & store
	res := c.ccLimitedChecker.CheckURL(ctx, url)
	ttl := c.cacheSettings.cacheExpirationInterval
	if res.Status != Ok {
		ttl = c.cacheSettings.retryFailedAfter
	}
	if res.Status != Dropped {
		//c.cache.Set(url, res, cache.DefaultExpiration)
		c.ristrettoCache.SetWithTTL(url, res, 1, ttl)
	}
	return res
}

func (c *CachedURLChecker) shouldTakeCachedResult(res *URLCheckResult) bool {
	return res.Status == Ok ||
		res.Status == Skipped ||
		time.Now().Unix() <= res.FetchedAtEpochSeconds+c.retryFailedAfterSeconds
}
