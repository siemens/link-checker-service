// Copyright 2020 Siemens AG
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
// SPDX-License-Identifier: MPL-2.0
package infrastructure

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/viper"

	netUrl "net/url"

	"github.com/go-resty/resty/v2"
)

const defaultMaxRedirectsCount = 15
const defaultTimeoutSeconds = 10
const defaultUserAgent = "lcs/0.9"
const defaultBrowserUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/85.0.4183.83 Safari/537.36"
const defaultAcceptHeader = "*/*"

// URLCheckResult is the internal struct to hold URL check results
type URLCheckResult struct {
	Status                URLCheckStatus
	Code                  int
	Error                 error
	FetchedAtEpochSeconds int64
	BodyPatternsFound     []string
	RemoteAddr            string
}

// BodyPatternConfig is unmarshalled from the configuration file
type BodyPatternConfig struct {
	Name  string
	Regex string
}

type bodyPattern struct {
	name    string
	pattern *regexp.Regexp
}

type urlCheckerSettings struct {
	ProxyURL              string
	MaxRedirectsCount     uint
	TimeoutSeconds        uint
	UserAgent             string
	BrowserUserAgent      string
	AcceptHeader          string
	SkipCertificateCheck  bool
	SearchForBodyPatterns bool
	BodyPatterns          []bodyPattern
	EnableRequestTracing  bool
}

// URLChecker interface that all layers should conform to
type URLChecker interface {
	CheckURL(ctx context.Context, url string) *URLCheckResult
}

// URLCheckerClient contains the HTTP/URL checking logic
type URLCheckerClient struct {
	client             *resty.Client
	clientWithoutProxy *resty.Client
	settings           urlCheckerSettings
}

// NewURLCheckerClient instantiates a new basic URL checking client
func NewURLCheckerClient() *URLCheckerClient {
	urlCheckerSettings := getURLCheckerSettings()
	urlCheckerSettingsNoProxy := urlCheckerSettings
	urlCheckerSettingsNoProxy.ProxyURL = ""

	return &URLCheckerClient{
		client:             buildClient(urlCheckerSettings),
		clientWithoutProxy: buildClient(urlCheckerSettingsNoProxy),
		settings:           urlCheckerSettings,
	}
}

func getURLCheckerSettings() urlCheckerSettings {
	s := urlCheckerSettings{
		ProxyURL:          "",
		MaxRedirectsCount: defaultMaxRedirectsCount, /*will be overwritten via the cobra default value*/
		TimeoutSeconds:    defaultTimeoutSeconds,    /*will be overwritten via the cobra default value*/
		UserAgent:         defaultUserAgent,
		BrowserUserAgent:  defaultBrowserUserAgent,
		AcceptHeader:      defaultAcceptHeader,
	}

	if proxyURL := viper.GetString("proxy"); proxyURL != "" {
		_, err := netUrl.Parse(proxyURL)
		if err != nil {
			log.Printf("Rejected proxyURL: %v", proxyURL)
		} else {
			log.Printf("URLCheckerClient is using a proxy: %v", proxyURL)
			s.ProxyURL = proxyURL
		}
	}

	s.MaxRedirectsCount = viper.GetUint("HTTPClient.maxRedirectsCount")
	s.TimeoutSeconds = viper.GetUint("HTTPClient.timeoutSeconds")
	if v := viper.GetString("HTTPClient.userAgent"); v != "" {
		s.UserAgent = v
	}
	if v := viper.GetString("HTTPClient.browserUserAgent"); v != "" {
		s.BrowserUserAgent = v
	}
	if v := viper.GetString("HTTPClient.acceptHeader"); v != "" {
		s.AcceptHeader = v
	}
	s.SkipCertificateCheck = viper.GetBool("HTTPClient.skipCertificateCheck")
	s.EnableRequestTracing = viper.GetBool("HTTPClient.enableRequestTracing")

	log.Printf("HTTP client MaxRedirectsCount: %v", s.MaxRedirectsCount)
	log.Printf("HTTP client TimeoutSeconds: %v", s.TimeoutSeconds)
	log.Printf("HTTP client UserAgent: %v", s.UserAgent)
	log.Printf("HTTP client BrowserUserAgent: %v", s.BrowserUserAgent)
	log.Printf("HTTP client AcceptHeader: %v", s.AcceptHeader)
	log.Printf("HTTP client SkipCertificateCheck: %v", s.SkipCertificateCheck)
	log.Printf("HTTP client EnableRequestTracing: %v", s.EnableRequestTracing)

	// advanced configuration feature: only configurable via the config file
	s.SearchForBodyPatterns = viper.GetBool("searchForBodyPatterns")

	if s.SearchForBodyPatterns {
		log.Printf("Will search for regex patterns found in HTTP response bodies")
		var configBodyPatterns []BodyPatternConfig
		// advanced configuration feature: only configurable via the config file
		if err := viper.UnmarshalKey("bodyPatterns", &configBodyPatterns); err == nil {
			for _, pattern := range configBodyPatterns {
				r := regexp.MustCompile(pattern.Regex)
				s.BodyPatterns = append(s.BodyPatterns, bodyPattern{
					name:    pattern.Name,
					pattern: r,
				})
				log.Printf("Body search pattern found. Name: '%v', Regex: '%v'", pattern.Name, pattern.Regex)
			}
		}
	}

	return s
}

// CheckURL checks a single URL
func (c *URLCheckerClient) CheckURL(ctx context.Context, url string) *URLCheckResult {
	res, shouldAbort := c.checkURL(ctx, url, c.client)
	if shouldAbort {
		return res
	}

	if res.Code == http.StatusBadGateway && c.settings.ProxyURL != "" {
		res, _ = c.checkURL(ctx, url, c.clientWithoutProxy)
		return res
	}

	return res
}

func normalizeAddressOf(input string) string {
	u, err := url.Parse(input)
	if err != nil {
		// bad urls will be handled later by the client
		return "<bad url>"
	}
	port:=u.Port()
	if port=="" {
		switch u.Scheme {
		case "http":
		 	port = "80"
		case "https":
			port = "443"
		}
	}

	return u.Host + ":" + port
}

func (c *URLCheckerClient) checkURL(ctx context.Context, urlToCheck string, client *resty.Client) (*URLCheckResult, bool) {
	select {
	case <-ctx.Done():
		return &URLCheckResult{
			Status:                Dropped,
			Code:                  CustomHTTPErrorCode,
			Error:                 fmt.Errorf("processing aborted"),
			FetchedAtEpochSeconds: time.Now().Unix(),
		}, true
	default:
		// do not block if not cancelled
	}

	remoteAddr := ""

	if c.settings.EnableRequestTracing {
		trace := &httptrace.ClientTrace{
			ConnectDone: func(network, _addr string, err error) {
				if err == nil {
					if addr, err := net.ResolveTCPAddr(network, normalizeAddressOf(urlToCheck)); err == nil {
						remoteAddr = addr.String()
					} else {
						log.Print(err)
					}
				}
			},
		}
		ctx = httptrace.WithClientTrace(ctx, trace)
	}

	response, err := client.R().
		SetHeader("Accept", c.settings.AcceptHeader).
		SetHeader("User-Agent", c.settings.UserAgent).
		SetContext(ctx).
		Head(urlToCheck)

	res := c.processResponse(urlToCheck, response, err)

	// Some sites don't allow robot user agents
	if res.Code == http.StatusForbidden {
		response, err = client.R().
			SetHeader("Accept", c.settings.AcceptHeader).
			SetHeader("User-Agent", c.settings.BrowserUserAgent).
			Head(urlToCheck)
		res = c.processResponse(urlToCheck, response, err)
	}

	var body string
	// some sites don't allow HEAD requests, try a GET
	if c.settings.SearchForBodyPatterns ||
		res.Code == http.StatusForbidden ||
		res.Code == http.StatusMethodNotAllowed ||
		res.Code == http.StatusServiceUnavailable ||
		res.Code == http.StatusNotFound /* e.g. www.tripadvisor.com */ {
		response, err = client.R().
			SetHeader("Accept", c.settings.AcceptHeader).
			// browser agent as last resort?
			Get(urlToCheck)
		res = c.processResponse(urlToCheck, response, err)
		if c.settings.SearchForBodyPatterns && response != nil {
			body = response.String()
		}
	}

	if c.settings.SearchForBodyPatterns {
		res = c.searchForBodyPatterns(res, body)
	}

	res.RemoteAddr = remoteAddr

	return res, false
}

func (c *URLCheckerClient) processResponse(url string, response *resty.Response, err error) *URLCheckResult {
	nowEpoch := time.Now().Unix()

	// some browser-optimized cache-controlled CDN sites return an empty body if browser doesn't re-request
	if /*errored*/ err != nil &&
		/*but there's a response*/ response != nil && response.RawResponse != nil &&
		/*and the response is ok*/ response.RawResponse.StatusCode == http.StatusOK {

		// then, interpret the result as ok
		return &URLCheckResult{
			Status:                Ok,
			Code:                  http.StatusOK,
			FetchedAtEpochSeconds: nowEpoch,
			BodyPatternsFound:     []string{},
		}
	}

	if err != nil || response == nil {
		code := CustomHTTPErrorCode /*as there's no available status in this case*/
		msg := "no error specified"
		if err != nil {
			msg = strings.ToLower(err.Error())
		}

		// proxies can misbehave. classify them as "bad gateway"
		if strings.Contains(msg, "bad gateway") ||
			strings.Contains(msg, "timeout") ||
			strings.Contains(msg, "deadline") {
			code = http.StatusBadGateway
		}
		return &URLCheckResult{
			Status:                Broken,
			Code:                  code,
			Error:                 err,
			FetchedAtEpochSeconds: nowEpoch,
			BodyPatternsFound:     []string{},
		}
	}

	statusCode := response.StatusCode()

	if statusCode >= 300 {
		return &URLCheckResult{
			Status:                Broken,
			Code:                  statusCode,
			Error:                 fmt.Errorf("%v status on url '%v'", statusCode, url),
			FetchedAtEpochSeconds: nowEpoch,
			BodyPatternsFound:     []string{},
		}
	}

	return &URLCheckResult{
		Status:                Ok,
		Code:                  statusCode,
		FetchedAtEpochSeconds: nowEpoch,
		BodyPatternsFound:     []string{},
	}
}

// to do
func (c *URLCheckerClient) searchForBodyPatterns(res *URLCheckResult, body string) *URLCheckResult {
	for _, pattern := range c.settings.BodyPatterns {
		if pattern.pattern.MatchString(body) {
			res.BodyPatternsFound = append(res.BodyPatternsFound, pattern.name)
		}
	}
	return res
}

func buildClient(settings urlCheckerSettings) *resty.Client {
	client := resty.New()
	client.SetTimeout(time.Second * time.Duration(settings.TimeoutSeconds))
	client.SetCloseConnection(true)
	client.SetRedirectPolicy(resty.FlexibleRedirectPolicy(defaultMaxRedirectsCount))
	if settings.ProxyURL != "" {
		client.SetProxy(settings.ProxyURL)
	}
	if settings.SkipCertificateCheck {
		client.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	}
	return client
}
