// Copyright 2020 Siemens AG
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
// SPDX-License-Identifier: MPL-2.0
package infrastructure

import (
	"context"
	"fmt"
	"log"
	"time"

	"golang.org/x/time/rate"

	"github.com/spf13/viper"

	"github.com/platinummonkey/go-concurrency-limits/core"
	"github.com/platinummonkey/go-concurrency-limits/limiter"
	"github.com/platinummonkey/go-concurrency-limits/strategy"
)

const defaultMaxConcurrentRequests = 256

// CustomHTTPErrorCode is a custom error code to be able to recognize it externally
// see also: https://www.iana.org/assignments/http-status-codes/http-status-codes.xhtml
//           https://en.wikipedia.org/wiki/List_of_HTTP_status_codes
const CustomHTTPErrorCode = 528

// CCLimitedURLChecker is a concurrency-limited wrapper around a URLCheckerClient
type CCLimitedURLChecker struct {
	guard  core.Limiter
	client *DomainRateLimitedChecker
}

// NewCCLimitedURLChecker instantiates a new concurrency-limited URL checker
func NewCCLimitedURLChecker() *CCLimitedURLChecker {
	limitStrategy := strategy.NewSimpleStrategy(getMaxConcurrentRequests())

	defaultLimiter, err := limiter.NewDefaultLimiterWithDefaults(
		"example_blocking_limit",
		limitStrategy,
		nil, // limit.BuiltinLimitLogger{}
		core.EmptyMetricRegistryInstance,
	)
	if err != nil {
		log.Fatalf("Error creating limiter err=%v\n", err)
	}
	guard := limiter.NewBlockingLimiter(defaultLimiter, 0, nil /*logger*/)
	ratePerSecond := getDomainRatePerSecond()
	client := NewDomainRateLimitedChecker(ratePerSecond)
	return &CCLimitedURLChecker{
		guard:  guard,
		client: client,
	}
}

func getDomainRatePerSecond() rate.Limit {
	var ratePerSecond rate.Limit = 0
	if r := viper.GetFloat64("requestsPerSecondPerDomain"); r > 0 {
		ratePerSecond = rate.Limit(r)
	}
	return ratePerSecond
}

func getMaxConcurrentRequests() int {
	maxConcurrency := viper.GetUint("maxConcurrentHTTPRequests")
	if maxConcurrency > 0 {
		log.Printf("CCLimitedURLChecker is using max HTTP concurrency of %v", maxConcurrency)
	}
	return defaultMaxConcurrentRequests
}

// CheckURL checks the desired URL
func (r *CCLimitedURLChecker) CheckURL(ctx context.Context, url string) *URLCheckResult {
	if ctx == nil {
		ctx = context.Background()
	}
	return r.checkURL(ctx, url)
}

func (r *CCLimitedURLChecker) checkURL(ctx context.Context, url string) *URLCheckResult {
	nowEpoch := time.Now().Unix()

	token, ok := r.guard.Acquire(ctx)
	if !ok {
		// short circuited - no need to try
		log.Printf("guarded request short circuited for url '%v'\n", url)
		if token != nil {
			token.OnDropped()
		}
		return &URLCheckResult{
			Status:                Dropped,
			Code:                  CustomHTTPErrorCode,
			Error:                 fmt.Errorf("short circuited request"),
			FetchedAtEpochSeconds: nowEpoch,
		}
	}

	resultChannel := make(chan *URLCheckResult, 0)
	// allow for cancellation -> run in a goroutine
	go func() {
		// try making the request
		resultChannel <- r.client.CheckURL(ctx, url)
	}()

	select {
	case res := <-resultChannel:
		token.OnSuccess()
		return res
	case <-ctx.Done():
		// client probably disconnected
		token.OnDropped()
		return &URLCheckResult{
			Status:                Dropped,
			Code:                  CustomHTTPErrorCode,
			Error:                 fmt.Errorf("cancelled request"),
			FetchedAtEpochSeconds: nowEpoch,
		}
	}
}
