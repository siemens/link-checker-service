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

func setUpViperTestConfiguration() {
	viper.SetEnvPrefix("LCS")
	viper.Set("proxy", os.Getenv("LCS_PROXY"))
	viper.Set("HTTPClient.timeoutSeconds", uint(15))
	viper.Set("HTTPClient.maxRedirectsCount", uint(15))
	viper.Set("searchForBodyPatterns", false)
	patterns := []struct {
		Name  string
		Regex string
	}{
		{"google", "google"},
	}
	viper.Set("bodyPatterns", patterns)
}
