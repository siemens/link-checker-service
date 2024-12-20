// Copyright 2020-2024 Siemens AG
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
// SPDX-License-Identifier: MPL-2.0

package server

import (
	"context"
	"io"
	"math"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/gin-contrib/cors"
	"github.com/gobwas/glob"

	ginzerolog "github.com/dn365/gin-zerolog"

	"github.com/MicahParks/keyfunc"
	ginGwt "github.com/appleboy/gin-jwt/v2"
	jwtv4 "github.com/golang-jwt/jwt/v4"
	"github.com/ulule/limiter/v3"
	gm "github.com/ulule/limiter/v3/drivers/middleware/gin"
	"github.com/ulule/limiter/v3/drivers/store/memory"

	"github.com/gin-gonic/gin"
	"github.com/siemens/link-checker-service/infrastructure"
)

// to do: parameterized
const totalRequestDeadlineTimeoutSecondsPerURL = 15
const totalRequestDeadlineTimeoutSeconds = 300
const largeRequestLoggingThreshold = 200

// JWTValidationOptions configures authentication via JWT validation
type JWTValidationOptions struct {
	PrivKeyFile      string
	PubKeyFile       string
	SigningAlgorithm string
	JwksUrl          string
}

// Options configures the web service instance
type Options struct {
	CORSOrigins           []string
	IPRateLimit           string
	MaxURLsInRequest      uint
	DisableRequestLogging bool
	DomainBlacklistGlobs  []string
	BindAddress           string
	JWTValidationOptions  *JWTValidationOptions
}

// Server starts an instance of the link checker service
type Server struct {
	server               *gin.Engine
	options              *Options
	urlChecker           *infrastructure.CachedURLChecker
	domainBlacklistGlobs []glob.Glob
}

// NewServerWithOptions creates a new server instance with custom options
func NewServerWithOptions(options *Options) Server {
	server := Server{
		server:               configureGin(options),
		urlChecker:           infrastructure.NewCachedURLChecker(),
		options:              options,
		domainBlacklistGlobs: precompileGlobs(options.DomainBlacklistGlobs),
	}
	server.setupRoutes()
	return server
}

func precompileGlobs(globs []string) []glob.Glob {

	if len(globs) == 0 {
		return nil
	}

	var res []glob.Glob
	for _, pattern := range globs {
		// see if more complex pattern handling needed
		// glob.MustCompile(pattern, '.','/')
		// see also https://github.com/gobwas/glob
		res = append(res, glob.MustCompile(pattern))
	}

	return res
}

func configureGin(options *Options) *gin.Engine {
	e := gin.New()
	e.Use(gin.Recovery())

	if options.DisableRequestLogging {
		log.Info().Msg("Disabling request logging")
		return e
	}

	e.Use(ginzerolog.Logger("gin"))

	return e
}

// NewServer creates a new server instance
func NewServer() Server {
	return NewServerWithOptions(&Options{})
}

// Detail exposes a *gin.Engine router for testing purposes
func (s *Server) Detail() *gin.Engine {
	return s.server
}

// Run starts the service instance (binds a port)
// set the PORT environment variable for a different port to bind at
func (s *Server) Run() {
	log.Info().Msgf("Go version: %s\n", runtime.Version())
	log.Info().Msgf("GOMAXPROCS: %v", runtime.GOMAXPROCS(-1))
	log.Info().Msgf("Instance ID: %v", infrastructure.GetInstanceId())
	var err error
	if s.options.BindAddress != "" {
		// custom bind address, e.g. 0.0.0.0:4444
		err = s.server.Run(s.options.BindAddress)
	} else {
		// default behavior: listen and serve on 0.0.0.0:${PORT:-8080}
		err = s.server.Run()
	}
	if err != nil {
		log.Fatal().Err(err).Msg("Could not start the server")
	}
}

func (s *Server) setupRoutes() {
	s.setUpCORS()

	if s.options.MaxURLsInRequest > 0 {
		log.Info().Msgf("Max URLs per request: %v", s.options.MaxURLsInRequest)
	}

	checkURLsRoutes := s.server.Group("/checkUrls")
	statsRoutes := s.server.Group("/stats")

	s.setUpRateLimiting(checkURLsRoutes)

	if s.options.JWTValidationOptions != nil {
		s.setUpJWTValidation(checkURLsRoutes, statsRoutes)
	}

	checkURLsRoutes.POST("", s.checkURLs)
	checkURLsRoutes.POST("/stream", s.checkURLsStream)

	s.server.GET("/version", s.getVersion)

	statsRoutes.GET("", s.getStats)
	statsRoutes.GET("/domains", s.getDomainStats)

	s.server.GET("/livez", s.getHealthStatus)
	s.server.GET("/readyz", s.getHealthStatus)
}

func (s *Server) checkURLs(c *gin.Context) {
	infrastructure.GlobalStats().OnIncomingRequest()
	request, abort := s.parseURLCheckRequestOrAbort(c, false)
	if abort {
		return
	}
	response := s.checkURLsInParallel(c.Request.Context(), request)
	if response.Result == "aborted" {
		return
	}
	// just mirror the request for now
	c.JSON(http.StatusOK, response)
}

func (s *Server) checkURLsInParallel(ctx context.Context, request CheckURLsRequest) *CheckURLsResponse {
	resultURLs := make([]URLStatusResponse, 0)

	urls, deadline, resultChannel, doneChannel := s.setUpAsyncURLCheck(ctx, request)

	for {
		select {
		case <-deadline.C:
			log.Info().Msg("Deadline reached, returning a partial result.")
			return &CheckURLsResponse{
				Urls:   urls.allResultsDeduplicated(resultURLs),
				Result: "partial",
			}

		case <-ctx.Done():
			log.Info().Msg("Client disconnected, aborting processing.")
			return &CheckURLsResponse{
				Urls:   urls.allResultsDeduplicated(resultURLs),
				Result: "aborted",
			}

		case urlStatus := <-resultChannel:
			resultURLs = append(resultURLs, urlStatus)

		case <-doneChannel:
			return &CheckURLsResponse{
				Urls:   urls.allResultsDeduplicated(resultURLs),
				Result: "complete",
			}
		}
	}
}

func (s *Server) checkURLsStream(c *gin.Context) {
	infrastructure.GlobalStats().OnIncomingRequest()
	infrastructure.GlobalStats().OnIncomingStreamRequest()

	request, abort := s.parseURLCheckRequestOrAbort(c, true)
	if abort {
		return
	}

	ctx := c.Request.Context()
	urls, deadline, resultChannel, doneChannel := s.setUpAsyncURLCheck(ctx, request)
	closeNotify := c.Writer.CloseNotify()

	// callback returns false on end of processing
	c.Stream(func(w io.Writer) bool {
		select {
		case <-deadline.C:
			log.Info().Msg("Deadline reached, aborting the stream.")
			return false
		case <-ctx.Done():
			log.Info().Msg("Client disconnected, aborting the stream.")
			return false

		case <-closeNotify:
			log.Info().Msg("Client closed the connection, aborting the stream.")
			return false

		case urlStatus := <-resultChannel:
			for _, duplicatedURLStatus := range urls.deduplicatedResultFor(urlStatus) {
				c.JSON(http.StatusOK, duplicatedURLStatus)
				c.String(http.StatusOK, "\n")
				c.Writer.(http.Flusher).Flush()
			}
			return true

		case <-doneChannel:
			return false
		}
	})
}

func (s *Server) parseURLCheckRequestOrAbort(c *gin.Context, stream bool) (CheckURLsRequest, bool) {
	var request CheckURLsRequest
	err := c.BindJSON(&request)
	if err != nil {
		c.String(http.StatusBadRequest, "Could not parse json: %v", err.Error())
		return CheckURLsRequest{}, true
	}
	count := len(request.Urls)
	log.Info().
		Int("count", count).
		Bool("large", count > largeRequestLoggingThreshold).
		Bool("stream", stream).
		Msg("Link check request")

	if s.options.MaxURLsInRequest != 0 && uint(count) > s.options.MaxURLsInRequest {
		c.String(http.StatusRequestEntityTooLarge, "Number of URLs in request limit exceeded")
		return CheckURLsRequest{}, true
	}

	if uint(count) == 0 {
		c.String(http.StatusBadRequest, "No URLs in request body")
		return CheckURLsRequest{}, true
	}
	return request, false
}

func (s *Server) setUpCORS() {
	if s.options.CORSOrigins != nil && len(s.options.CORSOrigins) > 0 {
		corsConfig := cors.DefaultConfig()
		corsConfig.AllowOrigins = s.options.CORSOrigins
		corsConfig.AllowMethods = []string{"POST", "GET", "OPTIONS"}
		corsConfig.AllowHeaders = []string{"last-event-id", "Authorization"}
		corsConfig.AllowCredentials = true
		log.Info().Msgf("Using CORS headers: %v", corsConfig.AllowOrigins)
		s.server.Use(cors.New(corsConfig))
	}
}

func (s *Server) setUpAsyncURLCheck(ctx context.Context, request CheckURLsRequest) (*deduplicator, *time.Timer, chan URLStatusResponse, chan struct{}) {
	urls := deduplicateURLs(request.Urls)
	count := len(urls.toCheck)
	duplicateCount := len(urls.toDuplicate)
	if duplicateCount > 0 {
		log.Info().Msgf("Duplicate URLs found: %v", duplicateCount)
	}

	deadline := time.NewTimer(time.Second * time.Duration(int64(math.Max(float64(totalRequestDeadlineTimeoutSecondsPerURL*count), float64(totalRequestDeadlineTimeoutSeconds)))))
	wg := sync.WaitGroup{}
	resultChannel := make(chan URLStatusResponse)
	doneChannel := make(chan struct{})

	// fire off all url requests in parallel
	// and let the rate limiter and cache do the work
	wg.Add(count)
	for _, u := range urls.toCheck {
		go func(url URLRequest) {
			defer wg.Done()
			response := s.checkURL(ctx, url)
			urls.onResponse(&response)
			resultChannel <- response
		}(u)
	}

	go func() {
		defer close(resultChannel)
		wg.Wait()
		doneChannel <- struct{}{}
	}()
	return urls, deadline, resultChannel, doneChannel
}

func (s *Server) checkURL(ctx context.Context, url URLRequest) URLStatusResponse {
	if s.domainBlacklistGlobs != nil && s.isBlacklisted(url) {
		return urlBlacklisted(url)
	}

	checkResult := s.urlChecker.CheckURL(ctx, url.URL)
	errorString := ""
	if checkResult.Error != nil {
		errorString = checkResult.Error.Error()
	}
	urlStatus := URLStatusResponse{
		URLRequest:            url,
		HTTPStatus:            checkResult.Code,
		Status:                strings.ToLower(checkResult.Status.String()), //-transform didn't work
		Error:                 errorString,
		FetchedAtEpochSeconds: checkResult.FetchedAtEpochSeconds,
		BodyPatternsFound:     checkResult.BodyPatternsFound,
		RemoteAddr:            checkResult.RemoteAddr,
		CheckTrace:            translateCheckerTrace(checkResult.CheckerTrace),
		ElapsedMs:             checkResult.ElapsedMs,
	}
	return urlStatus
}

func translateCheckerTrace(trace []infrastructure.URLCheckerPluginTrace) []URLCheckTraceResponse {
	var res []URLCheckTraceResponse
	for _, traceRes := range trace {
		res = append(res, URLCheckTraceResponse{
			Name:      traceRes.Name,
			Code:      traceRes.Code,
			ElapsedMs: traceRes.ElapsedMs,
			Error:     traceRes.Error,
		})
	}
	return res
}

func urlBlacklisted(url URLRequest) URLStatusResponse {
	infrastructure.GlobalStats().OnLinkSkipped(infrastructure.DomainOf(url.URL))
	return URLStatusResponse{
		URLRequest:            url,
		HTTPStatus:            infrastructure.CustomHTTPErrorCode,
		Status:                strings.ToLower(infrastructure.Skipped.String()), //-transform didn't work
		Error:                 "url was blacklisted",
		FetchedAtEpochSeconds: time.Now().Unix(),
		BodyPatternsFound:     []string{},
	}
}

func (s *Server) isBlacklisted(input URLRequest) bool {
	// use the domain without the port
	domain := infrastructure.DomainOf(input.URL)
	for _, g := range s.domainBlacklistGlobs {
		if g.Match(domain) {
			return true
		}
	}
	return false
}

func (s *Server) setUpJWTValidation(routerGroups ...*gin.RouterGroup) {
	if s.options.JWTValidationOptions == nil {
		log.Fatal().Msg("JWT Validation not set up correctly")
	}

	log.Info().Msg("Using JWT Validation")
	log.Info().Msgf("  PrivKeyFile: %v", s.options.JWTValidationOptions.PrivKeyFile)
	log.Info().Msgf("  PubKeyFile: %v", s.options.JWTValidationOptions.PubKeyFile)
	log.Info().Msgf("  SigningAlgorithm: %v", s.options.JWTValidationOptions.SigningAlgorithm)

	// the jwt middleware
	middleware, err := ginGwt.New(&ginGwt.GinJWTMiddleware{
		KeyFunc:          tryGetJwksKeyFunc(s.options.JWTValidationOptions.JwksUrl),
		PrivKeyFile:      s.options.JWTValidationOptions.PrivKeyFile,
		PubKeyFile:       s.options.JWTValidationOptions.PubKeyFile,
		SigningAlgorithm: s.options.JWTValidationOptions.SigningAlgorithm,
		HTTPStatusMessageFunc: func(e error, c *gin.Context) string {
			log.Error().Err(e).Msg("Token validation error")
			return "Token validation error: unauthorized"
		},
	})

	if err != nil {
		log.Fatal().Err(err).Msg("JWT Error")
	}

	for _, routerGroup := range routerGroups {
		routerGroup.Use(middleware.MiddlewareFunc())
	}
}

func tryGetJwksKeyFunc(jwksURL string) func(token *jwtv4.Token) (interface{}, error) {
	kf, err := keyfunc.Get(jwksURL, keyfunc.Options{
		// to do: configurable
		RefreshInterval: 1 * time.Hour,
	})
	if err != nil {
		return nil
	}
	log.Info().Msgf("JWKS configured with the url: %s", jwksURL)
	return kf.Keyfunc
}

func (s *Server) setUpRateLimiting(routerGroup *gin.RouterGroup) {
	if s.options.IPRateLimit == "" {
		log.Info().Msgf("Not using IP rate limiting")
		return
	}

	// see https://github.com/ulule/limiter
	rate, err := limiter.NewRateFromFormatted(s.options.IPRateLimit)
	if err != nil {
		log.Info().Msgf("Not using IP rate limiting: %v", err)
		return
	}

	log.Info().Msgf("Using IP rate limiting with a specified rate of %v", s.options.IPRateLimit)

	store := memory.NewStore()

	middleware := gm.NewMiddleware(limiter.New(store, rate))
	s.server.ForwardedByClientIP = true
	routerGroup.Use(middleware)
}

func (s *Server) getVersion(c *gin.Context) {
	c.String(http.StatusOK, infrastructure.BinaryVersion())
}

// always healthy for now
func (s *Server) getHealthStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "UP",
	})
}

const instanceIdHeader = "X-INSTANCE-ID"
const runningSinceHeader = "X-RUNNING-SINCE"

func (s *Server) getStats(c *gin.Context) {
	c.Header(instanceIdHeader, infrastructure.GetInstanceId())
	c.Header(runningSinceHeader, infrastructure.GetRunningSince())
	c.JSON(http.StatusOK, infrastructure.GlobalStats().GetStats())
}

func (s *Server) getDomainStats(c *gin.Context) {
	c.Header(instanceIdHeader, infrastructure.GetInstanceId())
	c.Header(runningSinceHeader, infrastructure.GetRunningSince())
	c.JSON(http.StatusOK, infrastructure.GlobalStats().GetDomainStats())
}
