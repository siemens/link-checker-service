// Copyright 2020 Siemens AG
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
// SPDX-License-Identifier: MPL-2.0
package server

import (
	"context"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gobwas/glob"
	"github.com/ulule/limiter/v3"
	gm "github.com/ulule/limiter/v3/drivers/middleware/gin"
	"github.com/ulule/limiter/v3/drivers/store/memory"

	"github.com/gin-contrib/cors"

	"github.com/gin-gonic/gin"
	"github.com/siemens/link-checker-service/infrastructure"
)

// to do: parameterized
const totalRequestDeadlineTimeoutSecondsPerURL = 15
const totalRequestDeadlineTimeoutSeconds = 300
const largeRequestLoggingThreshold = 200

// Options configures the web service instance
type Options struct {
	CORSOrigins           []string
	IPRateLimit           string
	MaxURLsInRequest      uint
	DisableRequestLogging bool
	DomainBlacklistGlobs  []string
	BindAddress           string
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

	if globs == nil || len(globs) == 0 {
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
	if options.DisableRequestLogging {
		e := gin.New()
		e.Use(gin.Recovery())
		log.Println("Disabling request logging")
		return e
	}
	return gin.Default()
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
	var err error
	if s.options.BindAddress != "" {
		// custom bind address, e.g. 0.0.0.0:4444
		err = s.server.Run(s.options.BindAddress)
	} else {
		// default behavior: listen and serve on 0.0.0.0:${PORT:-8080}
		err = s.server.Run()
	}
	if err != nil {
		log.Fatalf("Could not start the server: %v", err)
	}
}

func (s *Server) setupRoutes() {
	s.setUpCORS()
	s.setUpRateLimiting()

	if s.options.MaxURLsInRequest > 0 {
		log.Printf("Max URLs per request: %v", s.options.MaxURLsInRequest)
	}

	s.server.POST("/checkUrls", s.checkURLs)
	s.server.POST("/checkUrls/stream", s.checkURLsStream)

	s.server.GET("/version", s.getVersion)
}

func (s *Server) checkURLs(c *gin.Context) {
	request, abort := s.parseURLCheckRequestOrAbort(c)
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
			log.Printf("Deadline reached, returning a partial result.")
			return &CheckURLsResponse{
				Urls:   urls.allResultsDeduplicated(resultURLs),
				Result: "partial",
			}

		case <-ctx.Done():
			log.Printf("Client disconnected, aborting processing.")
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
	request, abort := s.parseURLCheckRequestOrAbort(c)
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
			log.Printf("Deadline reached, aborting the stream.")
			return false
		case <-ctx.Done():
			log.Printf("Client disconnected, aborting the stream.")
			return false

		case <-closeNotify:
			log.Printf("Client closed the connection, aborting the stream.")
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

func (s *Server) parseURLCheckRequestOrAbort(c *gin.Context) (CheckURLsRequest, bool) {
	var request CheckURLsRequest
	err := c.BindJSON(&request)
	if err != nil {
		c.String(http.StatusBadRequest, "Could not parse json: %v", err.Error())
		return CheckURLsRequest{}, true
	}
	count := len(request.Urls)
	if count > largeRequestLoggingThreshold {
		log.Printf("Large request: %v urls", count)
	}

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
		corsConfig.AllowMethods = []string{"POST"}
		corsConfig.AllowHeaders = []string{"last-event-id"}
		log.Printf("Using CORS headers: %v", corsConfig.AllowOrigins)
		s.server.Use(cors.New(corsConfig))
	}
}

func (s *Server) setUpAsyncURLCheck(ctx context.Context, request CheckURLsRequest) (*deduplicator, *time.Timer, chan URLStatusResponse, chan struct{}) {
	urls := deduplicateURLs(request.Urls)
	count := len(urls.toCheck)
	duplicateCount := len(urls.toDuplicate)
	if duplicateCount > 0 {
		log.Printf("Duplicate URLs found: %v", duplicateCount)
	}

	deadline := time.NewTimer(time.Second * time.Duration(int64(math.Max(float64(totalRequestDeadlineTimeoutSecondsPerURL*count), float64(totalRequestDeadlineTimeoutSeconds)))))
	wg := sync.WaitGroup{}
	resultChannel := make(chan URLStatusResponse, 0)
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
	}
	return urlStatus
}

func urlBlacklisted(url URLRequest) URLStatusResponse {
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
	u, err := url.Parse(input.URL)
	if err != nil {
		return false
	}
	for _, g := range s.domainBlacklistGlobs {
		if g.Match(u.Host) {
			return true
		}
	}
	return false
}

func (s *Server) setUpRateLimiting() {
	if s.options.IPRateLimit == "" {
		log.Printf("Not using IP rate limiting")
		return
	}

	// see https://github.com/ulule/limiter
	rate, err := limiter.NewRateFromFormatted(s.options.IPRateLimit)
	if err != nil {
		log.Printf("Not using IP rate limiting: %v", err)
		return
	}

	log.Printf("Using IP rate limiting with a specified rate of %v", s.options.IPRateLimit)

	store := memory.NewStore()

	middleware := gm.NewMiddleware(limiter.New(store, rate))
	s.server.ForwardedByClientIP = true
	s.server.Use(middleware)
}

func (s *Server) getVersion(c *gin.Context) {
	c.String(http.StatusOK, infrastructure.BinaryVersion())
}
