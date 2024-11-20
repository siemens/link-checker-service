// Copyright 2020-2024 Siemens AG
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
// SPDX-License-Identifier: MPL-2.0

package infrastructure

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"golang.org/x/time/rate"
)

// DomainRateLimitedChecker is a domain-rate-limited URLCheckerClient wrapper
type DomainRateLimitedChecker struct {
	domains       sync.Map
	ratePerSecond rate.Limit
	checker       *URLCheckerClient
}

// NewDomainRateLimitedChecker Creates a new domain-rate-limited URLCheckerClient instance
func NewDomainRateLimitedChecker(ratePerSecond rate.Limit) *DomainRateLimitedChecker {
	if ratePerSecond > 0 {
		log.Info().Msgf("Limiting amount of requests per domain to %v/s", ratePerSecond)
	}
	return &DomainRateLimitedChecker{
		domains:       sync.Map{},
		ratePerSecond: ratePerSecond,
		checker:       NewURLCheckerClient(),
	}
}

// CheckURL checks the desired URL applying rate limits per domain
func (c *DomainRateLimitedChecker) CheckURL(ctx context.Context, url string) *URLCheckResult {
	// if there's no limiting, just check
	if c.ratePerSecond == 0 {
		return c.checker.CheckURL(ctx, url)
	}

	// limit per domain
	key := DomainOf(url)
	var limiter *rate.Limiter
	if limiterInstance, ok := c.domains.Load(key); !ok {
		limiter = rate.NewLimiter(c.ratePerSecond /*per second*/, 1 /*burst*/)
	} else {
		limiter = limiterInstance.(*rate.Limiter)
	}
	if err := limiter.Wait(ctx); err != nil {
		nowEpoch := time.Now().Unix()

		// some browser-optimized cache-controlled CDN sites return an empty body if browser doesn't re-request
		return &URLCheckResult{
			Status:                Dropped,
			Code:                  CustomHTTPErrorCode,
			Error:                 fmt.Errorf("domain rate limiter aborted: %v", err),
			FetchedAtEpochSeconds: nowEpoch,
			BodyPatternsFound:     []string{},
		}
	}
	return c.checker.CheckURL(ctx, url)
}
