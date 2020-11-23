// Copyright 2020 Siemens AG
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
// SPDX-License-Identifier: MPL-2.0
package infrastructure

import (
	"context"
	"net/http"
	"os"
	"testing"

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
	res := NewURLCheckerClient().CheckURL(context.Background(), "https://google.com")
	assert.Nil(t, res.Error)
	assert.Equal(t, http.StatusOK, res.Code)
	assert.Len(t, res.BodyPatternsFound, 1)
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
	viper.Set("searchForBodyPatterns", false)
	viper.Set("urlCheckerPlugins", []string{})
	patterns := []struct {
		Name  string
		Regex string
	}{
		{"google", "google"},
	}
	viper.Set("bodyPatterns", patterns)
}

func TestDefaultURLCheckerClientPlugins(t *testing.T) {
	// the default is a single URL client-based checker
	setUpViperTestConfiguration()
	c := NewURLCheckerClient()
	assert.Equal(t, []string{"urlcheck"}, c.settings.UrlCheckerPlugins)

	// setting multiple checkers is possible
	viper.Set("urlCheckerPlugins", []string{"urlcheck", "_always_ok", "urlcheck"})
	c = NewURLCheckerClient()
	assert.Equal(t, []string{"urlcheck", "_always_ok", "urlcheck"}, c.settings.UrlCheckerPlugins)

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
		assert.Equal(t, []string{"urlcheck", "urlcheck-noproxy"}, c.settings.UrlCheckerPlugins)
	})
}

func TestCheckerSequenceMatters(t *testing.T) {
	setUpViperTestConfiguration()
	viper.Set("urlCheckerPlugins", []string{"_always_ok", "_always_bad"})
	res := NewURLCheckerClient().CheckURL(context.Background(), "http://lkasdjfasf.com/123987")
	assert.Equal(t, Ok, res.Status, "the result should've been Ok as _always_ok comes first and aborts the chain")

	viper.Set("urlCheckerPlugins", []string{"_always_bad", "_always_ok"})
	res = NewURLCheckerClient().CheckURL(context.Background(), "http://lkasdjfasf.com/123987")
	assert.NotEqual(t, Ok, res.Status, "the result should not have been Ok as _always_bad comes first and aborts the chain")
}
