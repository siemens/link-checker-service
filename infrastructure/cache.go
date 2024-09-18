// Copyright 2020-2023 Siemens AG
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
// SPDX-License-Identifier: MPL-2.0

package infrastructure

import (
	"encoding/json"
	"time"

	"github.com/dgraph-io/ristretto"
	"github.com/patrickmn/go-cache"
)

type resultCache interface {
	Get(url string) (*URLCheckResult, bool)
	Set(url string, res *URLCheckResult)
}

type ristrettoCache struct {
	cache             *ristretto.Cache[string, *URLCheckResult]
	defaultExpiration time.Duration
}

func (c ristrettoCache) Set(url string, res *URLCheckResult) {
	c.cache.SetWithTTL(url, res, approxSizeOf(url, res), c.defaultExpiration)
}

func (c ristrettoCache) Get(url string) (*URLCheckResult, bool) {
	value, found := c.cache.Get(url)

	if found {
		return value, true
	}

	return nil, false
}

type defaultCache struct {
	cache *cache.Cache
}

func (c defaultCache) Set(url string, res *URLCheckResult) {
	c.cache.Set(url, res, cache.DefaultExpiration)
}

func (c defaultCache) Get(url string) (*URLCheckResult, bool) {
	value, found := c.cache.Get(url)

	if found {
		return value.(*URLCheckResult), true
	}

	return nil, false
}

func newCache(settings cacheSettings) resultCache {
	if settings.cacheUseRistretto {
		return newRistrettoCache(settings)
	}

	return newDefaultCache(settings)
}

func newRistrettoCache(settings cacheSettings) *ristrettoCache {
	// https://github.com/dgraph-io/ristretto#Config
	rc, err := ristretto.NewCache(&ristretto.Config[string, *URLCheckResult]{
		NumCounters: settings.cacheNumCounters, // number of keys to track frequency of (~10x max links)
		MaxCost:     settings.cacheMaxSize,     // maximum cost of cache (in bytes)
		BufferItems: 64,                        // number of keys per Get buffer: as recommended
	})
	if err != nil {
		panic(err)
	}
	return &ristrettoCache{
		cache:             rc,
		defaultExpiration: settings.cacheExpirationInterval,
	}
}

func newDefaultCache(settings cacheSettings) *defaultCache {
	return &defaultCache{
		cache: cache.New(settings.cacheExpirationInterval, settings.cacheCleanupInterval),
	}
}

func approxSizeOf(key string, res *URLCheckResult) int64 {
	bytes, err := json.Marshal(res)
	if err != nil {
		return 512 + int64(len(key)) // any number should suffice - approximate calculation
	}
	blob := string(bytes)
	return int64(len(blob) + len(key))
}
