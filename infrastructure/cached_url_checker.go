// Copyright 2020-2021 Siemens AG
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
// SPDX-License-Identifier: MPL-2.0
package infrastructure

import (
	"context"
	"log"
	"time"

	"github.com/spf13/viper"
)

const defaultCacheExpirationInterval = 24 * time.Hour
const defaultCacheCleanupInterval = 48 * time.Hour
const defaultRetryFailedAfter = 30 * time.Second
const defaultCacheMaxSize int64 = 1e9
const defaultCacheNumCounters int64 = 10_000_000

// CachedURLChecker wraps a concurrency-limited URL checker
type CachedURLChecker struct {
	cache                   resultCache
	retryFailedAfterSeconds int64

	ccLimitedChecker *CCLimitedURLChecker
}

type cacheSettings struct {
	cacheUseRistretto       bool
	cacheExpirationInterval time.Duration
	cacheCleanupInterval    time.Duration
	cacheMaxSize            int64
	cacheNumCounters        int64
	retryFailedAfter        time.Duration
}

// NewCachedURLChecker creates a new cached URL checker instance
func NewCachedURLChecker() *CachedURLChecker {
	settings := fetchCachedURLCheckerSettings()

	checker := CachedURLChecker{
		cache:                   newCache(settings),
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

	cacheUseRistretto := viper.GetBool("cacheUseRistretto")
	log.Printf("cacheUseRistretto: %v", cacheUseRistretto)
	s.cacheUseRistretto = cacheUseRistretto

	cacheMaxSize := defaultCacheMaxSize
	if cms := viper.GetInt64("cacheMaxSize"); cms > 0 {
		cacheMaxSize = cms
	}
	s.cacheMaxSize = cacheMaxSize

	cacheNumCounters := defaultCacheNumCounters
	if cnc := viper.GetInt64("cacheNumCounters"); cnc > 0 {
		cacheNumCounters = cnc
	}
	s.cacheNumCounters = cacheNumCounters

	if cacheUseRistretto {
		log.Printf("cacheMaxSize: %v", cacheMaxSize)
		log.Printf("cacheNumCounters: %v", cacheNumCounters)
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
	res, found := c.cache.Get(url)

	if found && c.shouldTakeCachedResult(res) {
		// failures could have been temporary -> retry a URL after some time
		return res
	}

	// otherwise, do the check & store
	res = c.ccLimitedChecker.CheckURL(ctx, url)
	if res.Status != Dropped {
		c.cache.Set(url, res)
	}
	return res
}

func (c *CachedURLChecker) shouldTakeCachedResult(res *URLCheckResult) bool {
	return res.Status == Ok ||
		res.Status == Skipped ||
		time.Now().Unix() <= res.FetchedAtEpochSeconds+c.retryFailedAfterSeconds
}
