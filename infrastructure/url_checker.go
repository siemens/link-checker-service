// Copyright 2020-2023 Siemens AG
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
// SPDX-License-Identifier: MPL-2.0

package infrastructure

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptrace"

	"regexp"
	"strings"
	"time"

	"github.com/darren/gpac"

	"github.com/patrickmn/go-cache"

	"github.com/spf13/viper"

	netUrl "net/url"

	"github.com/go-resty/resty/v2"
)

const defaultLimitBodyToNBytes = 0
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
	CheckerTrace          []URLCheckerPluginTrace
	ElapsedMs             int64
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
	URLCheckerPlugins     []string
	PacScriptURL          string
	LimitBodyToNBytes     uint
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
	dnsCache           *cache.Cache
	checkerPlugins     []URLCheckerPlugin
	autoProxy          *gpac.Parser
}

// NewURLCheckerClient instantiates a new basic URL checking client
func NewURLCheckerClient() *URLCheckerClient {
	urlCheckerSettings := getURLCheckerSettings()

	c := &URLCheckerClient{
		settings: urlCheckerSettings,
		dnsCache: cache.New(defaultCacheExpirationInterval, defaultCacheCleanupInterval),
	}

	if c.settings.PacScriptURL != "" {
		c.autoProxy = parsePacScript(c.settings.PacScriptURL)
	}

	var checkers []URLCheckerPlugin

	// for now, a valid checker may be configured twice, for whatever reason
	for _, checkerName := range c.settings.URLCheckerPlugins {
		switch checkerName {
		case "urlcheck":
			// default client
			checkers = addChecker(checkers, newLocalURLChecker(c, "urlcheck", buildClient(urlCheckerSettings)))
			log.Println("Added the defaut URL checker")
		case "urlcheck-pac":
			if c.settings.PacScriptURL == "" {
				panic("Cannot instantiate a 'urlcheck-pac' checker without a proxy auto-config script configured")
			}
			checkers = addChecker(checkers, newLocalURLChecker(c, "urlcheck-pac", nil))
			log.Println("Added the PAC file based URL checker")
		case "urlcheck-noproxy":
			// if proxy is defined, add one without the proxy as fallback
			if urlCheckerSettings.ProxyURL == "" {
				panic("No point in adding a 'urlcheck-noproxy' checker, as no proxy URL is defined")
			}

			urlCheckerSettingsNoProxy := urlCheckerSettings
			urlCheckerSettingsNoProxy.ProxyURL = ""
			checkers = addChecker(checkers, newLocalURLChecker(c, "urlcheck-noproxy", buildClient(urlCheckerSettingsNoProxy)))
			log.Println("Added the URL checker that doesn't use a proxy")
		case "_ok_after_1s_on_delay.com":
			// fake client for testing
			checkers = addChecker(checkers, &fakeURLChecker{1 * time.Second, &URLCheckResult{
				Status:                Ok,
				Code:                  http.StatusOK,
				Error:                 nil,
				FetchedAtEpochSeconds: 0,
				BodyPatternsFound:     nil,
				RemoteAddr:            "",
			}, "_ok_after_1s_on_delay.com"})
			log.Println("Added the _always_ok checker")
		case "_always_ok":
			// fake client for testing
			checkers = addChecker(checkers, &fakeURLChecker{0, &URLCheckResult{
				Status:                Ok,
				Code:                  http.StatusOK,
				Error:                 nil,
				FetchedAtEpochSeconds: 0,
				BodyPatternsFound:     nil,
				RemoteAddr:            "",
			}, "_always_ok"})
			log.Println("Added the _always_ok checker")
		case "_always_bad":
			// fake client for testing
			checkers = addChecker(checkers, &fakeURLChecker{0, &URLCheckResult{
				Status:                Broken,
				Code:                  http.StatusInternalServerError,
				Error:                 fmt.Errorf("bad"),
				FetchedAtEpochSeconds: 0,
				BodyPatternsFound:     nil,
				RemoteAddr:            "",
			}, "_always_bad"})
			log.Println("Added the _always_bad checker")
		default:
			panic(fmt.Errorf("unknown checker: %v", checkerName))
		}
	}

	if len(checkers) == 0 {
		panic("Found no checker plugins. Please define one using '-p'")
	}

	c.checkerPlugins = checkers

	return c
}

func parsePacScript(scriptURL string) *gpac.Parser {
	res, err := resty.New().R().Get(scriptURL)
	if err != nil {
		panic(fmt.Errorf("could not fetch a PAC script from %v: %v", scriptURL, err.Error()))
	}
	log.Printf("Read PAC script from %v", scriptURL)
	script := string(res.Body())
	pac, err := gpac.New(script)
	if err != nil {
		panic(fmt.Errorf("could not parse the PAC script: %v", err.Error()))
	}
	return pac
}

func addChecker(checkers []URLCheckerPlugin, plugin URLCheckerPlugin) []URLCheckerPlugin {
	if plugin != nil {
		return append(checkers, plugin)
	}
	return checkers
}

func getURLCheckerSettings() urlCheckerSettings {
	s := urlCheckerSettings{
		ProxyURL:          "",
		MaxRedirectsCount: defaultMaxRedirectsCount, /*will be overwritten via the cobra default value*/
		TimeoutSeconds:    defaultTimeoutSeconds,    /*will be overwritten via the cobra default value*/
		UserAgent:         defaultUserAgent,
		BrowserUserAgent:  defaultBrowserUserAgent,
		AcceptHeader:      defaultAcceptHeader,
		LimitBodyToNBytes: defaultLimitBodyToNBytes,
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

	if pacScriptURL := viper.GetString("pacScriptURL"); pacScriptURL != "" {
		s.PacScriptURL = pacScriptURL
	}

	s.MaxRedirectsCount = viper.GetUint("HTTPClient.maxRedirectsCount")
	s.LimitBodyToNBytes = viper.GetUint("HTTPClient.limitBodyToNBytes")
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
	log.Printf("HTTP client LimitBodyToNBytes: %v", s.LimitBodyToNBytes)

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

	urlCheckerPlugins := []string{"urlcheck"}
	const urlCheckerPluginsKey = "urlCheckerPlugins"
	g := viper.GetStringSlice(urlCheckerPluginsKey)
	// empty string slice config creates a single slice with a "[]" -> fix
	if g != nil && !(len(g) == 1 && g[0] == "[]") && len(g) > 0 {
		urlCheckerPlugins = viper.GetStringSlice(urlCheckerPluginsKey)
	}
	s.URLCheckerPlugins = urlCheckerPlugins

	return s
}

type fakeURLChecker struct {
	delay        time.Duration
	alwaysReturn *URLCheckResult
	name         string
}

func (l *fakeURLChecker) Name() string {
	return l.name
}

func (l *fakeURLChecker) CheckURL(_ context.Context, url string, _ *URLCheckResult) (*URLCheckResult, bool) {
	if l.delay != 0 && strings.Contains(url, "delay.com") {
		time.Sleep(l.delay)
	}
	return l.alwaysReturn, true /* aborts the chain for now */
}

func newLocalURLChecker(c *URLCheckerClient, name string, client *resty.Client) *localURLChecker {
	return &localURLChecker{
		c:      c,
		client: client,
		name:   name,
	}
}

type localURLChecker struct {
	c      *URLCheckerClient
	client *resty.Client
	name   string
}

func (l *localURLChecker) Name() string {
	return l.name
}

func (l *localURLChecker) CheckURL(ctx context.Context, urlToCheck string, lastResult *URLCheckResult) (*URLCheckResult, bool) {
	if lastResult == nil || shouldRetryBasedOnStatus(lastResult.Code) {
		client := l.client
		if client == nil && l.c.settings.PacScriptURL != "" {
			client = l.autoSelectClientFor(urlToCheck)
		}
		if client == nil {
			panic("cannot instantiate a HTTP client. Please check the configuration")
		}
		GlobalStats().OnOutgoingRequest()
		res, err := l.c.checkURL(ctx, urlToCheck, client)
		onCheckResult(DomainOf(urlToCheck), res)
		return res, err
	}
	return lastResult, false
}

func onCheckResult(domain string, res *URLCheckResult) {
	s := GlobalStats()
	if res == nil {
		s.OnLinkErrored(domain)
		return
	}
	switch res.Status {
	case Ok:
		s.OnLinkOk(domain)
	case Broken:
		s.OnLinkBroken(domain, fmt.Sprintf("%v", res.Code))
	case Dropped:
		// handled in the drop handler
	case Skipped:
		// handled in an upper layer
		break
	}
}

func (l *localURLChecker) autoSelectClientFor(urlToCheck string) *resty.Client {
	tmpSettings := l.c.settings
	proxies, err := l.c.autoProxy.FindProxy(urlToCheck)
	if err == nil && len(proxies) > 0 {
		// choosing the first available proxy
		for _, proxy := range proxies {
			if proxy.Type == "PROXY" {
				tmpSettings.ProxyURL = proxies[0].URL()
				break
			}
		}
	} else {
		log.Printf("Could not find a proxy for %v", sanitizeUserLogInput(urlToCheck))
	}
	return buildClient(tmpSettings)
}

// URLCheckerPluginTrace is the internal struct to gather individual checker plugin stats
type URLCheckerPluginTrace struct {
	Name      string
	Code      int
	ElapsedMs int64
	Error     string
}

// CheckURL checks a single URL
func (c *URLCheckerClient) CheckURL(ctx context.Context, url string) *URLCheckResult {
	var lastRes *URLCheckResult = nil
	var checkerTrace []URLCheckerPluginTrace
	start := time.Now()
	errorMessage := ""

	for pos, currentChecker := range c.checkerPlugins {
		checkerStart := time.Now()
		res, shouldAbort := currentChecker.CheckURL(ctx, url, lastRes)
		checkerElapsed := time.Since(checkerStart)
		if res.Error != nil {
			errorMessage = res.Error.Error()
		}
		checkerTrace = append(checkerTrace, URLCheckerPluginTrace{
			Name:      currentChecker.Name(),
			Code:      res.Code,
			ElapsedMs: int64(checkerElapsed / time.Millisecond),
			Error:     errorMessage,
		})

		if pos == 0 && res == nil {
			panic("first checker should never return nil")
		}

		lastRes = res

		if shouldAbort || !shouldRetryBasedOnStatus(lastRes.Code) {
			break
		}
	}

	if lastRes != nil {
		lastRes.CheckerTrace = checkerTrace
		elapsed := time.Since(start)
		lastRes.ElapsedMs = int64(elapsed / time.Millisecond)
	}

	return lastRes
}

func normalizeAddressOf(input string) string {
	u, err := netUrl.Parse(input)
	if err != nil {
		// bad urls will be handled later by the client
		return "<bad url>"
	}
	port := u.Port()
	if port == "" {
		switch u.Scheme {
		case "http":
			port = "80"
		case "https":
			port = "443"
		}
		return u.Host + ":" + port
	}

	return u.Host
}

func (c *URLCheckerClient) checkURL(ctx context.Context, urlToCheck string, client *resty.Client) (*URLCheckResult, bool) {
	select {
	case <-ctx.Done():
		domain := DomainOf(urlToCheck)
		GlobalStats().OnLinkDropped(domain)
		return &URLCheckResult{
			Status:                Dropped,
			Code:                  CustomHTTPErrorCode,
			Error:                 fmt.Errorf("processing aborted"),
			FetchedAtEpochSeconds: time.Now().Unix(),
		}, true
	default:
		// do not block if not cancelled
	}

	addrToResolve := normalizeAddressOf(urlToCheck)
	remoteAddr := c.cachedRemoteAddr(addrToResolve)

	domain := DomainOf(urlToCheck)

	if c.settings.EnableRequestTracing &&
		remoteAddr == "" /*enable tracing only if remoteAddr hasn't been resolved yet */ {

		// remoteAddr must be captured from the encompassing scope in the closures below
		trace := &httptrace.ClientTrace{
			ConnectDone: func(network, _addr string, err error) {
				remoteAddr = c.resolveAndCacheTCPAddr(network, err, addrToResolve)
			},
			DNSDone: func(info httptrace.DNSDoneInfo) {
				if remoteAddr == "" {
					GlobalStats().OnDNSResolutionFailed(domain)
					// this may not be as precise as ConnectDone, thus skipping caching
					remoteAddr = getDNSAddressesAsString(info.Addrs)
				}
			},
		}
		ctx = httptrace.WithClientTrace(ctx, trace)
	}

	res := c.tryHeadRequestDefault(ctx, urlToCheck, client)
	res = c.tryHeadRequestAsBrowserIfForbidden(ctx, urlToCheck, client, res)
	res = c.tryGetRequestAndProcessResponseBody(ctx, urlToCheck, client, res)

	res.RemoteAddr = remoteAddr

	return res, false
}

func (c *URLCheckerClient) tryGetRequestAndProcessResponseBody(ctx context.Context, urlToCheck string, client *resty.Client, res *URLCheckResult) *URLCheckResult {
	var body string
	// some sites don't allow HEAD requests, try a GET
	if c.settings.SearchForBodyPatterns ||
		shouldRetryBasedOnStatus(res.Code) {
		response, err := client.R().
			SetHeader("Accept", c.settings.AcceptHeader).
			SetContext(ctx).
			SetDoNotParseResponse(true).
			SetHeader("User-Agent", c.settings.BrowserUserAgent).
			Get(urlToCheck)
		res = c.processResponse(urlToCheck, response, err)
		if c.settings.SearchForBodyPatterns && response != nil {
			body = c.limitedBody(response)
		}
	}

	if c.settings.SearchForBodyPatterns {
		res = c.searchForBodyPatterns(res, body)
	}
	return res
}

func shouldRetryBasedOnStatus(code int) bool {
	if code < 300 {
		return false
	}

	return code == http.StatusForbidden ||
		code == http.StatusMethodNotAllowed ||
		code == http.StatusServiceUnavailable ||
		code == http.StatusNotFound ||
		code == CustomHTTPErrorCode ||
		code == http.StatusGatewayTimeout ||
		code == http.StatusBadGateway ||
		code == http.StatusRequestTimeout
}

func (c *URLCheckerClient) tryHeadRequestDefault(ctx context.Context, urlToCheck string, client *resty.Client) *URLCheckResult {
	response, err := client.R().
		SetHeader("Accept", c.settings.AcceptHeader).
		SetHeader("User-Agent", c.settings.UserAgent).
		SetContext(ctx).
		Head(urlToCheck)

	res := c.processResponse(urlToCheck, response, err)
	return res
}

func (c *URLCheckerClient) resolveAndCacheTCPAddr(network string, err error, addrToResolve string) string {
	remoteAddr := ""
	domain := DomainOf(addrToResolve)
	if err == nil {
		if addr, err := net.ResolveTCPAddr(network, addrToResolve); err == nil {
			// this may be called multiple times: last invocation wins
			remoteAddr = addr.String()
			c.dnsCache.Set(addrToResolve, remoteAddr, defaultCacheExpirationInterval)
		} else {
			GlobalStats().OnDNSResolutionFailed(domain)
			c.dnsCache.Set(addrToResolve, "DNS resolution failed", defaultRetryFailedAfter)
			log.Printf("ERROR in resolveAndCacheTCPAddr: %v", err)
		}
	}
	return remoteAddr
}

func (c *URLCheckerClient) cachedRemoteAddr(addrToResolve string) string {
	remoteAddr := ""

	if resolved, found := c.dnsCache.Get(addrToResolve); found {
		remoteAddr = resolved.(string)
	}
	return remoteAddr
}

func getDNSAddressesAsString(addresses []net.IPAddr) string {
	var addr []string
	for _, a := range addresses {
		addr = append(addr, a.String())
	}

	return strings.Join(addr, ", ")
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

func (c *URLCheckerClient) searchForBodyPatterns(res *URLCheckResult, body string) *URLCheckResult {
	for _, pattern := range c.settings.BodyPatterns {
		if pattern.pattern.MatchString(body) {
			res.BodyPatternsFound = append(res.BodyPatternsFound, pattern.name)
		}
	}
	return res
}

func (c *URLCheckerClient) tryHeadRequestAsBrowserIfForbidden(ctx context.Context, urlToCheck string, client *resty.Client, res *URLCheckResult) *URLCheckResult {
	// Some sites don't allow robot user agents
	if res.Code == http.StatusForbidden {
		response, err := client.R().
			SetHeader("Accept", c.settings.AcceptHeader).
			SetHeader("User-Agent", c.settings.BrowserUserAgent).
			SetContext(ctx).
			Head(urlToCheck)
		res = c.processResponse(urlToCheck, response, err)
	}
	return res
}

func (c *URLCheckerClient) limitedBody(response *resty.Response) string {
	body := response.RawBody()
	if body == nil {
		return ""
	}
	defer body.Close()
	return safelyTrimmedStream(body, c.settings.LimitBodyToNBytes)
}

func safelyTrimmedStream(input io.Reader, limit uint) string {
	res := []byte{}
	if limit == 0 {
		b, err := io.ReadAll(input)
		if err != nil {
			if b != nil {
				res = b
			}
			return string(safelyTrimmedString(res, limit))
		}
		return string(b)
	}

	const bufferSize = 1024
	b := [bufferSize]byte{}
	bytesRead := 0
	for {
		n, err := input.Read(b[:])

		if err != nil {
			// first append bytes read so far
			res = append(res, b[:n]...)
			return string(safelyTrimmedString(res, limit))
		}

		res = append(res, b[:n]...)
		bytesRead += n

		if uint(bytesRead) >= limit {
			break
		}
	}
	return string(safelyTrimmedString(res, limit))
}

func safelyTrimmedString(s []byte, limit uint) []byte {
	if limit == 0 || len(s) <= int(limit) {
		return s
	}
	return s[:limit]
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
		// this is known to be insecure, thus protected via a configuration with a secure default
		client.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	}
	return client
}
