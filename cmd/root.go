// Copyright 2020-2023 Siemens AG
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/siemens/link-checker-service/infrastructure"

	"github.com/spf13/cobra"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

var cfgFile string

const (
	// service
	maxConcurrentHTTPRequestsKey  = "maxConcurrentHTTPRequests"
	cacheExpirationIntervalKey    = "cacheExpirationInterval"
	cacheCleanupIntervalKey       = "cacheCleanupInterval"
	cacheUseRistrettoKey          = "cacheUseRistretto"
	cacheMaxSizeKey               = "cacheMaxSize"
	cacheNumCountersKey           = "cacheNumCounters"
	retryFailedAfterKey           = "retryFailedAfter"
	maxURLsInRequestKey           = "maxURLsInRequest"
	requestsPerSecondPerDomainKey = "requestsPerSecondPerDomain"
	domainBlacklistGlobsKey       = "domainBlacklistGlobs"
	urlCheckerPluginsKey          = "urlCheckerPlugins"

	// HTTP client
	httpClientMapKey        = "HTTPClient."
	proxyKey                = "proxy"
	pacScriptURLKey         = "pacScriptURL"
	maxRedirectsCountKey    = "maxRedirectsCount"
	timeoutSecondsKey       = "timeoutSeconds"
	userAgentKey            = "userAgent"
	browserUserAgentKey     = "browserUserAgent"
	acceptHeaderKey         = "acceptHeader"
	skipCertificateCheckKey = "skipCertificateCheck"
	enableRequestTracingKey = "enableRequestTracing"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "link-checker-service",
	Short: "A web service to efficiently check for multiple broken URLs in batches",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	home := "$HOME"

	if homeString, err := homedir.Dir(); err == nil {
		if expandedHomeString, err := homedir.Expand(homeString); err == nil {
			home = expandedHomeString
		}
	}

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is "+filepath.Join(home, ".link-checker-service.toml)"))

	// HTTP client
	rootCmd.PersistentFlags().StringP(proxyKey, "", "", "HTTP client: proxy server to use, e.g. http://myproxy:8080")
	_ = viper.BindPFlag(proxyKey, rootCmd.PersistentFlags().Lookup(proxyKey))
	rootCmd.PersistentFlags().StringP(pacScriptURLKey, "", "", "HTTP client: PAC script URL, e.g. http://myproxy/proxy.pac")
	_ = viper.BindPFlag(pacScriptURLKey, rootCmd.PersistentFlags().Lookup(pacScriptURLKey))

	rootCmd.PersistentFlags().Uint(maxRedirectsCountKey, 15, "HTTP client: maximum number of redirects to follow")
	_ = viper.BindPFlag(httpClientMapKey+maxRedirectsCountKey, rootCmd.PersistentFlags().Lookup(maxRedirectsCountKey))
	rootCmd.PersistentFlags().Uint(timeoutSecondsKey, 30, "HTTP client: request timeout")
	_ = viper.BindPFlag(httpClientMapKey+timeoutSecondsKey, rootCmd.PersistentFlags().Lookup(timeoutSecondsKey))
	rootCmd.PersistentFlags().String(userAgentKey, "lcs/0.9", "HTTP client: user agent header to try first")
	_ = viper.BindPFlag(httpClientMapKey+userAgentKey, rootCmd.PersistentFlags().Lookup(userAgentKey))
	rootCmd.PersistentFlags().String(browserUserAgentKey, "", "HTTP client: custom alternative user agent to try if the default one is blocked")
	_ = viper.BindPFlag(httpClientMapKey+browserUserAgentKey, rootCmd.PersistentFlags().Lookup(browserUserAgentKey))
	rootCmd.PersistentFlags().String(acceptHeaderKey, "*/*", "HTTP client: accept header key to set")
	_ = viper.BindPFlag(httpClientMapKey+acceptHeaderKey, rootCmd.PersistentFlags().Lookup(acceptHeaderKey))
	rootCmd.PersistentFlags().Bool(skipCertificateCheckKey, false, "HTTP client: skip verifying server certificates")
	_ = viper.BindPFlag(httpClientMapKey+skipCertificateCheckKey, rootCmd.PersistentFlags().Lookup(skipCertificateCheckKey))
	rootCmd.PersistentFlags().Bool(enableRequestTracingKey, false, "HTTP client: enable request tracing")
	_ = viper.BindPFlag(httpClientMapKey+enableRequestTracingKey, rootCmd.PersistentFlags().Lookup(enableRequestTracingKey))
	// service
	rootCmd.PersistentFlags().UintP(maxConcurrentHTTPRequestsKey, "c", 256, "maximum number of total concurrent HTTP requests")
	_ = viper.BindPFlag(maxConcurrentHTTPRequestsKey, rootCmd.PersistentFlags().Lookup(maxConcurrentHTTPRequestsKey))

	// cache
	rootCmd.PersistentFlags().String(cacheExpirationIntervalKey, "24h", "Expire each URL check result after <interval> (in ns/us/ms/s/m/h)")
	_ = viper.BindPFlag(cacheExpirationIntervalKey, rootCmd.PersistentFlags().Lookup(cacheExpirationIntervalKey))
	rootCmd.PersistentFlags().String(cacheCleanupIntervalKey, "48h", "Interval between cache cleanups (in ns/us/ms/s/m/h)")
	_ = viper.BindPFlag(cacheCleanupIntervalKey, rootCmd.PersistentFlags().Lookup(cacheCleanupIntervalKey))
	rootCmd.PersistentFlags().Bool(cacheUseRistrettoKey, false, "Use a memory-bound cache (see the cacheMaxSize option)")
	_ = viper.BindPFlag(cacheUseRistrettoKey, rootCmd.PersistentFlags().Lookup(cacheUseRistrettoKey))
	rootCmd.PersistentFlags().Int64(cacheMaxSizeKey, 1000_000_000, "Approximage maximum cache size in bytes (when cacheUseRistretto enabled)")
	_ = viper.BindPFlag(cacheMaxSizeKey, rootCmd.PersistentFlags().Lookup(cacheMaxSizeKey))
	rootCmd.PersistentFlags().Int64(cacheNumCountersKey, 10_000_000, "Number of 4-bit access counters. Set at approx 10x max unique expected URLs (when cacheUseRistretto enabled)")
	_ = viper.BindPFlag(cacheNumCountersKey, rootCmd.PersistentFlags().Lookup(cacheNumCountersKey))

	rootCmd.PersistentFlags().String(retryFailedAfterKey, "30s", "If a URL check failed, e.g. intermittently, re-run it after <interval>  (in ns/us/ms/s/m/h)")
	_ = viper.BindPFlag(retryFailedAfterKey, rootCmd.PersistentFlags().Lookup(retryFailedAfterKey))
	rootCmd.PersistentFlags().UintP(maxURLsInRequestKey, "m", 0, "Maximum number URLs allowed per request")
	_ = viper.BindPFlag(maxURLsInRequestKey, rootCmd.PersistentFlags().Lookup(maxURLsInRequestKey))

	rootCmd.PersistentFlags().Float64(requestsPerSecondPerDomainKey, 10, "Maximum requests per second per domain")
	_ = viper.BindPFlag(requestsPerSecondPerDomainKey, rootCmd.PersistentFlags().Lookup(requestsPerSecondPerDomainKey))

	rootCmd.PersistentFlags().StringSliceP(domainBlacklistGlobsKey, "b", nil,
		"provide a list of domain wildcards to avoid checking, e.g. -b scaleyourc?de.* -b testdomain.com")
	_ = viper.BindPFlag(domainBlacklistGlobsKey, rootCmd.PersistentFlags().Lookup(domainBlacklistGlobsKey))

	rootCmd.PersistentFlags().StringSliceP(urlCheckerPluginsKey, "p", []string{"urlcheck"},
		"provide a list of URL checkers. Additionally, 'urlcheck-noproxy' can be used if a proxy is defined, and an additional check without a proxy makes sense. The argument sequence is the checker sequence.")
	_ = viper.BindPFlag(urlCheckerPluginsKey, rootCmd.PersistentFlags().Lookup(urlCheckerPluginsKey))

	SetUpViper()

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// SetUpViper configures environment variable and global flag handling
func SetUpViper() {
	viper.SetEnvPrefix("LCS")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".link-checker-service" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".link-checker-service")
	}

	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	_ = viper.ReadInConfig()
}

func echoConfig() {
	log.Printf("Link Checker Service. Version: %v", infrastructure.BinaryVersion())

	if viper.ConfigFileUsed() != "" {
		log.Printf("Using config file: %v", viper.ConfigFileUsed())
	}

	proxyURL := viper.GetString(proxyKey)
	if proxyURL != "" {
		log.Printf("Proxy: %v", proxyURL)
	}

	maxConcurrency := viper.GetUint(maxConcurrentHTTPRequestsKey)
	if maxConcurrency > 0 {
		log.Printf("Max HTTP concurrency: %v", maxConcurrency)
	}

	globCount := len(domainBlacklistGlobs)
	if globCount > 0 {
		log.Printf("Domain blacklist globs (%v): %v", globCount, domainBlacklistGlobs)
	}
}
