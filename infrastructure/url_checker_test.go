// Copyright 2020-2023 Siemens AG
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
// SPDX-License-Identifier: MPL-2.0

package infrastructure

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"

	"github.com/stretchr/testify/assert"
)

func TestBrokenUrls(t *testing.T) {
	setUpViperTestConfiguration()
	res := NewURLCheckerClient().CheckURL(context.Background(), "http://lkasdjfasf.com/123987")
	assert.NotNil(t, res.Error)
	assert.NotEqual(t, http.StatusOK, res.Code)
}

func TestOkUrls(t *testing.T) {
	setUpViperTestConfiguration()
	res := NewURLCheckerClient().CheckURL(context.Background(), "https://google.com")
	assert.Nil(t, res.Error)
	assert.Equal(t, http.StatusOK, res.Code)
	assert.Len(t, res.BodyPatternsFound, 0, "should not search for body patterns by default")
}

func TestSearchingForBodyPatterns(t *testing.T) {
	setUpViperTestConfiguration()
	viper.Set("searchForBodyPatterns", true)
	viper.Set("HTTPClient.limitBodyToNBytes", uint(0))
	res := NewURLCheckerClient().CheckURL(context.Background(), "https://google.com")
	assert.Nil(t, res.Error)
	assert.Equal(t, http.StatusOK, res.Code)
	require.Contains(t, res.BodyPatternsFound, "google")
	assert.Equal(t, "google", res.BodyPatternsFound[0], "should have found at least one mention of google")
}

func TestTracingRequests(t *testing.T) {
	setUpViperTestConfiguration()
	res := NewURLCheckerClient().CheckURL(context.Background(), "https://google.com")
	assert.Nil(t, res.Error)
	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, "", res.RemoteAddr)

	viper.Set("HTTPClient.enableRequestTracing", true)
	res = NewURLCheckerClient().CheckURL(context.Background(), "https://google.com")
	assert.Nil(t, res.Error)
	assert.Equal(t, http.StatusOK, res.Code)
	assert.NotEqual(t, "", res.RemoteAddr)
}

func TestNormalizingAddresses(t *testing.T) {
	assert.Equal(t, "a:80", normalizeAddressOf("http://a"))
	assert.Equal(t, "a:80", normalizeAddressOf("http://a:80"))
	assert.Equal(t, "a:81", normalizeAddressOf("http://a:81"))
	assert.Equal(t, "a:443", normalizeAddressOf("https://a:443"))
	assert.Equal(t, "a:443", normalizeAddressOf("https://a"))
}

func setUpViperTestConfiguration() {
	viper.SetEnvPrefix("LCS")
	viper.Set("proxy", os.Getenv("LCS_PROXY"))
	viper.Set("HTTPClient.timeoutSeconds", uint(15))
	viper.Set("HTTPClient.maxRedirectsCount", uint(15))
	viper.Set("HTTPClient.enableRequestTracing", false)
	viper.Set("HTTPClient.limitBodyToNBytes", uint(0))
	viper.Set("searchForBodyPatterns", false)
	viper.Set("urlCheckerPlugins", []string{})
	patterns := []struct {
		Name  string
		Regex string
	}{
		{"google", "google"},
		{"start-a", "start-a"},
		{"ab", "ab"},
	}
	viper.Set("bodyPatterns", patterns)
}

func TestDefaultURLCheckerClientPlugins(t *testing.T) {
	// the default is a single URL client-based checker
	setUpViperTestConfiguration()
	c := NewURLCheckerClient()
	assert.Equal(t, []string{"urlcheck"}, c.settings.URLCheckerPlugins)

	// setting multiple checkers is possible
	viper.Set("urlCheckerPlugins", []string{"urlcheck", "_always_ok", "urlcheck"})
	c = NewURLCheckerClient()
	assert.Equal(t, []string{"urlcheck", "_always_ok", "urlcheck"}, c.settings.URLCheckerPlugins)

	// unsetting the proxy disables the ability to add the noproxy client
	assert.Panics(t, func() {
		viper.Set("proxy", nil)
		viper.Set("urlCheckerPlugins", []string{"urlcheck", "urlcheck-noproxy"})
		NewURLCheckerClient()
	})

	// setting the proxy enables adding the noproxy client
	assert.NotPanics(t, func() {
		viper.Set("proxy", "http://proxy:1234")
		viper.Set("urlCheckerPlugins", []string{"urlcheck", "urlcheck-noproxy"})
		c = NewURLCheckerClient()
		assert.Equal(t, []string{"urlcheck", "urlcheck-noproxy"}, c.settings.URLCheckerPlugins)
	})

	// adding an unknown checker
	assert.Panics(t, func() {
		viper.Set("urlCheckerPlugins", []string{"urlcheck", "urlcheck-unknown"})
		NewURLCheckerClient()
	})
}

func TestCheckerSequenceMatters(t *testing.T) {
	setUpViperTestConfiguration()
	viper.Set("urlCheckerPlugins", []string{"_always_ok", "_always_bad"})
	res := NewURLCheckerClient().CheckURL(context.Background(), "http://lkasdjfasf.com/123987")
	assert.Equal(t, Ok, res.Status, "the result should've been Ok as _always_ok comes first and aborts the chain")
	assert.Equal(t, []URLCheckerPluginTrace{
		{Name: "_always_ok", Code: 200},
	}, res.CheckerTrace)

	viper.Set("urlCheckerPlugins", []string{"_always_bad", "_always_ok"})
	res = NewURLCheckerClient().CheckURL(context.Background(), "http://lkasdjfasf.com/123987")
	assert.NotEqual(t, Ok, res.Status, "the result should not have been Ok as _always_bad comes first and aborts the chain")
	assert.Len(t, res.CheckerTrace, 1)
	assert.Equal(t, "_always_bad", res.CheckerTrace[0].Name)
}

func TestResponseTimeout(t *testing.T) {
	setUpViperTestConfiguration()
	viper.Set("HTTPClient.timeoutSeconds", uint(1))
	start := time.Now()
	res := NewURLCheckerClient().CheckURL(context.Background(), "https://httpbin.org/delay/3")
	elapsed := time.Since(start)
	assert.True(t, elapsed < 3*time.Second, "the response should have been aborted after one second")
	assert.Greater(t, res.ElapsedMs, int64(1000), "at least 1 second must have passed")
	assert.Less(t, res.ElapsedMs, int64(3000), "at most 3 seconds must have passed")
	assert.NotNil(t, res.Error, "the response should have failed due to the abort")
	assert.NotEqual(t, http.StatusOK, res.Code)
}

func TestLimitingBodyReading(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintln(w,
			"start-"+
				strings.Repeat("a", 100)+
				strings.Repeat("b", 100))
	}))
	log.Println("Test server started at:", ts.URL)
	defer ts.Close()
	setUpViperTestConfiguration()
	viper.Set("searchForBodyPatterns", true)
	viper.Set("HTTPClient.limitBodyToNBytes", uint(100))
	res := NewURLCheckerClient().CheckURL(context.Background(), ts.URL)
	assert.Equal(t, http.StatusOK, res.Code)
	assert.Contains(t, res.BodyPatternsFound, "start-a")
	assert.NotContains(
		t,
		res.BodyPatternsFound,
		"ab",
		"the repeated 'b' part of the message should have not been processed",
	)
}
