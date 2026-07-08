// Copyright 2020-2024 Siemens AG
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
// SPDX-License-Identifier: MPL-2.0

package infrastructure

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"

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

func viperDuration(key string, fallback time.Duration) time.Duration {
	val := viper.GetString(key)
	d, err := time.ParseDuration(val)
	if err != nil {
		log.Info().Msgf("Ignoring %s %v -> %v (%v)", key, val, fallback, err)
		return fallback
	}
	log.Info().Msgf("%s: %v", key, val)
	return d
}

func fetchCachedURLCheckerSettings() cacheSettings {
	s := cacheSettings{}
	s.cacheExpirationInterval = viperDuration("cacheExpirationInterval", defaultCacheExpirationInterval)
	s.cacheCleanupInterval = viperDuration("cacheCleanupInterval", defaultCacheCleanupInterval)

	cacheUseRistretto := viper.GetBool("cacheUseRistretto")
	log.Info().Msgf("cacheUseRistretto: %v", cacheUseRistretto)
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
		log.Info().Msgf("cacheMaxSize: %v", cacheMaxSize)
		log.Info().Msgf("cacheNumCounters: %v", cacheNumCounters)
	}

	s.retryFailedAfter = viperDuration("retryFailedAfter", defaultRetryFailedAfter)
	return s
}

// CheckURL checks the desired URL
func (c *CachedURLChecker) CheckURL(ctx context.Context, url string) *URLCheckResult {
	res, found := c.cache.Get(url)

	if found && c.shouldTakeCachedResult(res) {
		GlobalStats().OnCacheHit()
		// failures could have been temporary -> retry a URL after some time
		return res
	}
	GlobalStats().OnCacheMiss()

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
