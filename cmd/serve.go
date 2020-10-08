// Copyright 2020 Siemens AG
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
// SPDX-License-Identifier: MPL-2.0
package cmd

import (
	s "github.com/siemens/link-checker-service/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var corsOrigins []string = nil

// IPRateLimit e.g. for 100 requests/minute: "100-M"
var IPRateLimit = ""
var maxURLsInRequest uint = 0
var disableRequestLogging = false
var domainBlacklistGlobs []string

const bindAddressKey = "bindAddress"

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Starts the link checker web server",
	Run: func(cmd *cobra.Command, args []string) {
		fetchConfig()
		echoConfig()
		server := s.NewServerWithOptions(&s.Options{
			CORSOrigins:           corsOrigins,
			IPRateLimit:           IPRateLimit,
			MaxURLsInRequest:      maxURLsInRequest,
			DisableRequestLogging: disableRequestLogging,
			DomainBlacklistGlobs:  domainBlacklistGlobs,
			BindAddress:           viper.GetString(bindAddressKey),
		})
		server.Run()
	},
}

func fetchConfig() {
	if corsOrigins == nil {
		if co := viper.GetStringSlice("corsOrigins"); co != nil {
			corsOrigins = co
		}
	}
	if IPRateLimit == "" {
		IPRateLimit = viper.GetString("IPRateLimit")
	}

	maxURLsInRequest = viper.GetUint(maxURLsInRequestKey)

	if viper.Get(domainBlacklistGlobsKey) != nil {
		g := viper.GetStringSlice(domainBlacklistGlobsKey)
		// empty string slice config creates a single slice with a "[]" -> fix
		if g != nil && !(len(g) == 1 && g[0] == "[]") {
			domainBlacklistGlobs = viper.GetStringSlice(domainBlacklistGlobsKey)
		}
	}
}

func init() {
	flags := serveCmd.Flags()
	flags.StringSliceVarP(&corsOrigins, "corsOrigins", "o", nil,
		"provide a list of CORS origins to enable CORS headers, e.g. '-o http://localhost:8080 -o http://localhost:8090")

	flags.StringP(bindAddressKey, "a", "",
		"bind to a different address other than `:8080`, i.e. 0.0.0.0:4444 or 127.0.0.1:4444")
	_ = viper.BindPFlag(bindAddressKey, flags.Lookup(bindAddressKey))

	flags.StringVar(&IPRateLimit, "IPRateLimit", "", "rate-limit requests from an IP. e.g. 5-S (5 per second), 1000-H (1000 per hour)")

	serveCmd.PersistentFlags().BoolVarP(&disableRequestLogging, "disableRequestLogging", "s", false, "disable request logging")

	rootCmd.AddCommand(serveCmd)
}
