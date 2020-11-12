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
var useJWTValidation = false

// IPRateLimit e.g. for 100 requests/minute: "100-M"
var IPRateLimit = ""
var maxURLsInRequest uint = 0
var disableRequestLogging = false
var domainBlacklistGlobs []string
var jwtValidationOptions *s.JWTValidationOptions = nil

const bindAddressKey = "bindAddress"
const useJWTValidationKey = "useJWTValidation"
const PrivKeyFileKey = "privKeyFile"
const PubKeyFileKey = "pubKeyFile"
const SigningAlgorithmKey = "signingAlgorithm"

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
			JWTValidationOptions:  jwtValidationOptions,
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

	if useJWTValidation {
		fetchJWTValidationConfig()
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

func fetchJWTValidationConfig() {
	jwtValidationOptions = &s.JWTValidationOptions{
		PrivKeyFile:      viper.GetString(PrivKeyFileKey),
		PubKeyFile:       viper.GetString(PubKeyFileKey),
		SigningAlgorithm: viper.GetString(SigningAlgorithmKey),
	}
}

func init() {
	flags := serveCmd.Flags()
	flags.StringSliceVarP(&corsOrigins, "corsOrigins", "o", nil,
		"provide a list of CORS origins to enable CORS headers, e.g. '-o http://localhost:8080 -o http://localhost:8090")

	flags.StringP(bindAddressKey, "a", "",
		"bind to a different address other than `:8080`, i.e. 0.0.0.0:4444 or 127.0.0.1:4444")
	_ = viper.BindPFlag(bindAddressKey, flags.Lookup(bindAddressKey))

	flags.BoolVar(&useJWTValidation, useJWTValidationKey, false,
		"use JWT validation")

	flags.String(PrivKeyFileKey, "dummy.priv.cer",
		"Provide a valid dummy private key certificate (work-around)")
	_ = viper.BindPFlag(PrivKeyFileKey, flags.Lookup(PrivKeyFileKey))

	flags.String(PubKeyFileKey, "public.cer",
		"Provide a valid public key to validate the JWT tokens against")
	_ = viper.BindPFlag(PubKeyFileKey, flags.Lookup(PubKeyFileKey))

	flags.String(SigningAlgorithmKey, "RS384",
		"Provide a valid public key to validate the JWT tokens against")
	_ = viper.BindPFlag(SigningAlgorithmKey, flags.Lookup(SigningAlgorithmKey))

	flags.StringVar(&IPRateLimit, "IPRateLimit", "", "rate-limit requests from an IP. e.g. 5-S (5 per second), 1000-H (1000 per hour)")

	serveCmd.PersistentFlags().BoolVarP(&disableRequestLogging, "disableRequestLogging", "s", false, "disable request logging")

	rootCmd.AddCommand(serveCmd)
}
