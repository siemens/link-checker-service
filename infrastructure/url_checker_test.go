// Copyright 2020-2024 Siemens AG
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
// SPDX-License-Identifier: MPL-2.0

package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

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

const startChunk = "start-"

var testStringToLimit = startChunk +
	strings.Repeat("a", 300) +
	strings.Repeat("b", 300)

func TestLimitingBodyReading(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintln(w,
			testStringToLimit)
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

func Test_safelyTrimmedStream(t *testing.T) {
	t.Run("limiting empty input produces empty string", func(t *testing.T) {
		assert.Equal(t, "", safelyTrimmedStream(streamOf(""), 10))
	})

	t.Run("non-empty input is not limited if no limit configured", func(t *testing.T) {
		assert.Equal(t, testStringToLimit, safelyTrimmedStream(streamOf(testStringToLimit), 0))
	})

	t.Run("limiting input to a size smaller than a chunk returns string of the limit length",
		func(t *testing.T) {
			assert.Equal(t, startChunk, safelyTrimmedStream(streamOf(testStringToLimit), uint(len(startChunk))))
		})

	t.Run("limiting input to a size larger than itself returns the original string",
		func(t *testing.T) {
			assert.Equal(t, testStringToLimit, safelyTrimmedStream(streamOf(testStringToLimit), 2000))
		})

	t.Run("limiting input to one byte results in one character",
		func(t *testing.T) {
			assert.Equal(t, 1, len(safelyTrimmedStream(streamOf(testStringToLimit), 1)))
		})

	t.Run("limiting input larger than the the buffer (1kB) to a limit larger than the buffer trims the input",
		func(t *testing.T) {
			assert.Equal(t, 1200, len(safelyTrimmedStream(streamOf(
				strings.Repeat(testStringToLimit, 2),
			), 1200)))
		})

	t.Run("trimming the errored stream returns the input processed", func(t *testing.T) {
		assert.Equal(t, "abc", safelyTrimmedStream(faultyReaderOf(
			"abc,d", 3,
		), 10))
	})

	t.Run("untrimmed errored stream returns the input processed", func(t *testing.T) {
		assert.Equal(t, "abc", safelyTrimmedStream(faultyReaderOf(
			"abc,d", 3,
		), 0))
	})
}

type faultyReader struct {
	input   string
	errorAt int
}

func (f *faultyReader) Read(p []byte) (int, error) {
	for i := 0; i < f.errorAt; i++ {
		p[i] = f.input[i]
	}
	return f.errorAt, errors.New("expected fault")
}

func faultyReaderOf(s string, i int) io.Reader {
	return &faultyReader{
		input:   s,
		errorAt: i,
	}
}

func streamOf(s string) io.Reader {
	return strings.NewReader(s)
}
