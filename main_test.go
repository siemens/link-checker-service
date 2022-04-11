// Copyright 2020-2022 Siemens AG
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/siemens/link-checker-service/infrastructure"

	"github.com/gin-gonic/gin"

	"github.com/spf13/viper"

	"github.com/siemens/link-checker-service/server"
	"github.com/stretchr/testify/assert"
)

const firstRequest = `
	{
		"urls": [
			{
				"url":"https://google.com",
				"context": "1"
			},
			{
				"url":"https://ashasdfdfkjhdf.com/kajhsd",
				"context": "2"
			}
		]
	}
	`

const secondRequest = `
	{
		"urls": [
			{
				"url":"https://bing.com",
				"context": "1"
			},
			{
				"url":"https://asdlfasdgisd.com/akdsjfsd",
				"context": "2"
			},
			{
				"url":"https://google.com",
				"context": "3"
			},
			{
				"url":"https://ashdfkjasdfhdf.com/kajhsd",
				"context": "4"
			}
		]
	}
	`

const endpoint = "/checkUrls"
const statsEndpoint = "/stats"
const streamingEndpoint = "/checkUrls/stream"

func TestCommonCheckUrlsUsage(t *testing.T) {
	setUpViperTestConfiguration()

	testServer := server.NewServer()
	router := testServer.Detail()

	// first request
	req, _ := http.NewRequest("POST", endpoint, strings.NewReader(firstRequest))
	response := fireAndParseRequest(t, router, req)
	assert.Len(t, response.Urls, 2)

	// first response
	okResponse1 := responseContaining("google", response)
	assert.Equal(t, "1", okResponse1.Context)
	assert.Equal(t, "", okResponse1.Error)
	assert.Greater(t, okResponse1.ElapsedMs, int64(0))

	// sleep for a bit
	time.Sleep(2 * time.Second)

	// second request
	req, _ = http.NewRequest("POST", endpoint, strings.NewReader(secondRequest))
	response = fireAndParseRequest(t, router, req)
	assert.Len(t, response.Urls, 4)
	// google.com
	okResponse2 := responseContaining("google", response)
	assert.Equal(t, "3", okResponse2.Context)
	assert.Equal(t, okResponse2.FetchedAtEpochSeconds, okResponse1.FetchedAtEpochSeconds, "the response should have been cached")

	newResponse := responseContaining("bing", response)
	assert.Equal(t, "1", newResponse.Context)
	assert.Greater(t, newResponse.FetchedAtEpochSeconds, okResponse1.FetchedAtEpochSeconds, "the new url should have been checked later than the cached one")

	assert.Greater(t, infrastructure.GlobalStats().GetDomainStats().DomainStats["google.com"].Ok, int64(0))
}

func responseContaining(urlSubstring string, response server.CheckURLsResponse) server.URLStatusResponse {
	for _, response := range response.Urls {
		if strings.Contains(response.URL, urlSubstring) {
			return response
		}
	}
	return server.URLStatusResponse{}
}

func fireAndParseRequest(t *testing.T, router *gin.Engine, req *http.Request) server.CheckURLsResponse {
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	response := unmarshalCheckURLsResponse(t, w)
	return response
}

func setUpViperTestConfiguration() {
	viper.SetEnvPrefix("LCS")
	viper.Set("proxy", os.Getenv("LCS_PROXY"))
	viper.Set("cacheUseRistretto", false)
}

func TestCORS(t *testing.T) {
	setUpViperTestConfiguration()
	const okOrigin = "http://localhost:8080"
	const wrongOrigin = "http://localhost:80"
	// start with CORS configuration
	testServer := server.NewServerWithOptions(&server.Options{
		CORSOrigins: []string{
			okOrigin,
		},
	})
	router := testServer.Detail()

	// no Origin header
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", endpoint, strings.NewReader(firstRequest))
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code, "the call without an origin header should have succeeded")

	// with a correct Origin header
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", endpoint, strings.NewReader(firstRequest))
	req.Header.Add("Origin", okOrigin)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code, "correct origin should be allowed by CORS")

	// with an incorrect Origin header
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", endpoint, strings.NewReader(firstRequest))
	req.Header.Add("Origin", wrongOrigin)
	router.ServeHTTP(w, req)
	assert.NotEqual(t, http.StatusOK, w.Code, "incorrect origin header should be blocked by CORS")

	// start without a CORS configuration
	testServer = server.NewServer()
	router = testServer.Detail()
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", endpoint, strings.NewReader(firstRequest))
	req.Header.Add("Origin", wrongOrigin)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code, "default server should allow any origin")
}

func TestRateLimiting(t *testing.T) {
	setUpViperTestConfiguration()
	// start a rate-limited server
	testServer := server.NewServerWithOptions(&server.Options{
		IPRateLimit: "10-S",
	})
	router := testServer.Detail()

	// cache the first request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", endpoint, strings.NewReader(firstRequest))
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code, "first call should have succeeded")

	for r := 0; r < 20; r++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", endpoint, strings.NewReader(firstRequest))
		router.ServeHTTP(w, req)
		if r > 10 {
			assert.NotEqual(t, http.StatusOK, w.Code, "calls above 10 within a second should have failed")
		}
	}

	// start a rate-unlimited server
	testServer = server.NewServerWithOptions(&server.Options{
		IPRateLimit: "",
	})
	router = testServer.Detail()

	// all requests should succeed
	for r := 0; r < 21; r++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", endpoint, strings.NewReader(firstRequest))
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code, "all requests should have succeeded")
	}
}

func TestPayloadLimiting(t *testing.T) {
	setUpViperTestConfiguration()
	// start a rate-limited server
	testServer := server.NewServerWithOptions(&server.Options{
		MaxURLsInRequest: 2,
	})
	router := testServer.Detail()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", endpoint, strings.NewReader(secondRequest))
	router.ServeHTTP(w, req)
	assert.NotEqual(t, http.StatusOK, w.Code, "4 urls should not have been allowed")
}

func TestUrlBlacklisting(t *testing.T) {
	setUpViperTestConfiguration()
	testServer := server.NewServerWithOptions(&server.Options{
		DomainBlacklistGlobs: []string{
			"test?atter*.*",
		},
	})
	router := testServer.Detail()

	request := `
	{
		"urls": [
			{
				"url":"https://testpattern.com",
				"context": "1"
			},
			{
				"url":"https://google.com",
				"context": "2"
			}
		]
	}
	`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", endpoint, strings.NewReader(request))
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	response := unmarshalCheckURLsResponse(t, w)
	assert.Len(t, response.Urls, 2)

	sort.Slice(response.Urls, func(i, j int) bool {
		return response.Urls[i].Context < response.Urls[j].Context
	})

	assert.Contains(t, response.Urls[0].Status, "skip")
	assert.Contains(t, response.Urls[1].Status, "ok")
}

func TestBadRequests(t *testing.T) {
	setUpViperTestConfiguration()
	testServer := server.NewServer()
	router := testServer.Detail()

	// bad route
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	router.ServeHTTP(w, req)
	assert.NotEqual(t, http.StatusOK, w.Code, "request to / should not have worked")

	// json parsing
	badRequestBody := `<xml>not json</xml>`
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", endpoint, strings.NewReader(badRequestBody))
	router.ServeHTTP(w, req)
	assert.NotEqual(t, http.StatusOK, w.Code, "bad post body should not have worked")

	// wrong method
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", endpoint, strings.NewReader(firstRequest))
	router.ServeHTTP(w, req)
	assert.NotEqual(t, http.StatusOK, w.Code, "GET with the correct request body should not have worked")
}

func TestDuplicateURLs(t *testing.T) {
	setUpViperTestConfiguration()
	viper.Set("maxConcurrentHTTPRequests", 1)
	testServer := server.NewServer()
	router := testServer.Detail()

	request := `
	{
		"urls": [
			{
				"url":"https://google.com",
				"context": "1"
			},
			{
				"url":"https://google.com  ",
				"context": "2"
			},
			{
				"url":"https://google.com/",
				"context": "3"
			}
		]
	}
	`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", endpoint, strings.NewReader(request))
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	response := unmarshalCheckURLsResponse(t, w)
	assert.Len(t, response.Urls, 3)
	assert.Equal(t, "ok", response.Urls[0].Status)
	assert.Equal(t, "ok", response.Urls[1].Status)
	assert.Equal(t, "ok", response.Urls[2].Status)
	assert.NotEqual(t, response.Urls[0].Context, response.Urls[1].Context, "context should have been preserved")
	assert.NotEqual(t, response.Urls[2].Context, response.Urls[1].Context, "context should have been preserved")
	assert.NotEqual(t, response.Urls[0].URL, response.Urls[1].URL, "original URL should have been preserved")
	assert.NotEqual(t, response.Urls[2].URL, response.Urls[1].URL, "original URL should have been preserved")
}

func TestBadResultsAreRecheckedAfterGracePeriod(t *testing.T) {
	infrastructure.ResetGlobalStats()

	setUpViperTestConfiguration()
	viper.Set("retryFailedAfter", "30s")
	testServer := server.NewServer()
	router := testServer.Detail()

	response1, response2 := fireTwoConsecutiveBadURLRequests(t, router)

	assert.Equal(
		t,
		int64(1),
		infrastructure.GlobalStats().GetStats().OutgoingRequests,
		"there should have been only one call",
	)

	// assert
	assert.Equal(
		t,
		response2.Urls[0].FetchedAtEpochSeconds,
		response1.Urls[0].FetchedAtEpochSeconds,
		"the result should have been cached",
	)

	viper.Set("retryFailedAfter", "0s")
	testServer = server.NewServer()
	router = testServer.Detail()

	response1, response2 = fireTwoConsecutiveBadURLRequests(t, router)

	// assert
	assert.Greater(
		t,
		response2.Urls[0].FetchedAtEpochSeconds,
		response1.Urls[0].FetchedAtEpochSeconds,
		"the result should have re-fetched, as retryFailedAfter=0s grace period is configured",
	)
}

func TestRistrettoCache(t *testing.T) {
	setUpViperTestConfiguration()
	viper.Set("cacheCleanupInterval", "1m")
	viper.Set("cacheUseRistretto", true)
	viper.Set("cacheMaxSize", 10 /*bytes, unrealistic*/)
	testServer := server.NewServer()
	router := testServer.Detail()

	response1, response2 := fireTwoConsecutiveOkRequests(t, router)

	// assert
	assert.Greater(
		t,
		response2.Urls[0].FetchedAtEpochSeconds,
		response1.Urls[0].FetchedAtEpochSeconds,
		"the result should not have been cached, as the max cost should have been exceeded",
	)

	// now, set the maximum size higher
	viper.Set("cacheMaxSize", 10_000_000)
	testServer = server.NewServer()
	router = testServer.Detail()

	response1, response2 = fireTwoConsecutiveOkRequests(t, router)

	// assert
	assert.Equal(
		t,
		response2.Urls[0].FetchedAtEpochSeconds,
		response1.Urls[0].FetchedAtEpochSeconds,
		"the result should have been cached, as the cache now has a sufficient maximum size",
	)
}

func fireTwoConsecutiveOkRequests(t *testing.T, router *gin.Engine) (server.CheckURLsResponse, server.CheckURLsResponse) {
	const alwaysOkRequest = `
	{
		"urls": [
			{
				"url":"https://google.com",
				"context": "1"
			}
		]
	}
	`

	// first request
	w := requestCheck(alwaysOkRequest, router)
	assert.Equal(t, http.StatusOK, w.Code)
	response1 := unmarshalCheckURLsResponse(t, w)
	assert.Len(t, response1.Urls, 1)
	assert.Equal(t, response1.Urls[0].Status, "ok")

	// sleep a bit
	time.Sleep(1 * time.Second)

	// second request
	w = requestCheck(alwaysOkRequest, router)
	assert.Equal(t, http.StatusOK, w.Code)
	response2 := unmarshalCheckURLsResponse(t, w)
	assert.Len(t, response2.Urls, 1)
	assert.Equal(t, response2.Urls[0].Status, "ok")

	return response1, response2
}

func fireTwoConsecutiveBadURLRequests(t *testing.T, router *gin.Engine) (server.CheckURLsResponse, server.CheckURLsResponse) {
	request := `
	{
		"urls": [
			{
				"url":"https://ajdhfsadflkjhasdf.com",
				"context": "0"
			}
		]
	}
	`

	// first request
	w := requestCheck(request, router)
	assert.Equal(t, http.StatusOK, w.Code)
	response1 := unmarshalCheckURLsResponse(t, w)
	assert.Len(t, response1.Urls, 1)
	assert.NotEqual(t, response1.Urls[0].Status, "ok")

	// sleep a bit
	time.Sleep(1 * time.Second)

	// second request
	w = requestCheck(request, router)
	assert.Equal(t, http.StatusOK, w.Code)
	response2 := unmarshalCheckURLsResponse(t, w)
	assert.Len(t, response2.Urls, 1)
	assert.NotEqual(t, response2.Urls[0].Status, "ok")
	return response1, response2
}

func requestCheck(request string, router *gin.Engine) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", endpoint, strings.NewReader(request))
	router.ServeHTTP(w, req)
	return w
}

func unmarshalCheckURLsResponse(t *testing.T, w *httptest.ResponseRecorder) server.CheckURLsResponse {
	response := server.CheckURLsResponse{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err, "unmarshalling the response should have worked")
	return response
}

func TestEmptyRequestsShouldFail(t *testing.T) {
	setUpViperTestConfiguration()
	testServer := server.NewServer()
	router := testServer.Detail()

	// async
	w := newCloseNotifyRecorder()
	req := httptest.NewRequest("POST", streamingEndpoint, strings.NewReader(`{"urls": []}`))
	router.ServeHTTP(w, req)
	body := w.Body.String()

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, strings.ToLower(body), "no")

	// sync
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req)

	body = w.Body.String()

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, strings.ToLower(body), "no")

	// empty urls
	req = httptest.NewRequest("POST", streamingEndpoint, strings.NewReader(`{}`))
	w2 = httptest.NewRecorder()
	router.ServeHTTP(w2, req)

	body = w.Body.String()

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, strings.ToLower(body), "no")
}

func TestStreamingResponse(t *testing.T) {
	setUpViperTestConfiguration()
	testServer := server.NewServer()
	router := testServer.Detail()

	w := newCloseNotifyRecorder()
	req := httptest.NewRequest("POST", streamingEndpoint, strings.NewReader(firstRequest))
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	if http.StatusOK != w.Code {
		// "didn't continue -> abort
		t.Fail()
		return
	}

	body := w.Body.String()

	// both contexts:
	assert.Contains(t, body, "\"1\"")
	assert.Contains(t, body, "\"2\"")
}

func TestJWTAuthentication(t *testing.T) {
	setUpViperTestConfiguration()
	pubKey, privKey, priv := createTestCertificates()
	testServer := server.NewServerWithOptions(&server.Options{
		JWTValidationOptions: &server.JWTValidationOptions{
			PrivKeyFile:      privKey,
			PubKeyFile:       pubKey,
			SigningAlgorithm: "RS384",
		},
	})
	router := testServer.Detail()

	// /version is not authenticated
	w := httptest.NewRecorder()
	versionReq, _ := http.NewRequest("GET", "/version", nil)
	router.ServeHTTP(w, versionReq)
	assert.Equal(t, http.StatusOK, w.Code, "the version endpoint should not be authenticated")

	// missing authentication
	w = httptest.NewRecorder()
	req, _ := http.NewRequest("POST", endpoint, strings.NewReader(firstRequest))
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code, "the call without a bearer token should fail")

	// /stats is authenticated
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", statsEndpoint, strings.NewReader(firstRequest))
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code, "the call to /stats without a bearer token should fail")

	// correct token
	token, _ := createJWTToken(priv)
	req, _ = http.NewRequest("POST", endpoint, strings.NewReader(firstRequest))
	req.Header.Add("Authorization", "Bearer "+token)
	response := fireAndParseRequest(t, router, req)
	assert.Len(t, response.Urls, 2)

	// bad token
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", endpoint, strings.NewReader(firstRequest))
	req.Header.Add("Authorization", "Bearer !bad!"+token)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code, "A bad JWT token should not have been valid")

	// cleanup
	_ = os.Remove(privKey)
	_ = os.Remove(pubKey)
}

func TestQuickResultsShouldArriveFirst(t *testing.T) {
	if runtime.GOMAXPROCS(-1) < 2 {
		t.Skip("To avoid sporadic failure, running this only when sufficient parallelism is given")
		return
	}
	setUpViperTestConfiguration()
	// first
	viper.Set("urlCheckerPlugins", []string{"_ok_after_1s_on_delay.com"})
	testServer := server.NewServer()
	router := testServer.Detail()

	// async
	w := newCloseNotifyRecorder()
	// first request should be delayed, and thus come second in the response
	req := httptest.NewRequest("POST", streamingEndpoint, strings.NewReader(
		strings.ReplaceAll(firstRequest, "google", "delay")))
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	if http.StatusOK != w.Code {
		// "didn't continue -> abort
		t.Fail()
		return
	}

	body := w.Body.String()
	responses := parseStreamingResponses(t, body)
	assert.Len(t, responses, 2)
	assert.NotContains(t, responses[0].URL, "delay")
	assert.Contains(t, responses[1].URL, "delay")
}

func parseStreamingResponses(t *testing.T, body string) []*server.URLStatusResponse {
	var res []*server.URLStatusResponse
	for _, line := range strings.Split(body, "\n") {
		if line != "" {
			response := server.URLStatusResponse{}
			err := json.Unmarshal([]byte(line), &response)
			assert.NoError(t, err, "unmarshalling the response should have worked")
			res = append(res, &response)
		}
	}
	return res
}
