// Copyright 2020-2024 Siemens AG
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
// SPDX-License-Identifier: MPL-2.0

// Package impersonate provides browser-mimicking HTTP headers and TLS configuration
// for the link-checker-service using only the standard library crypto/tls.
//
// It does not attempt exact TLS fingerprint matching (that would require a
// crypto/tls fork like utls). Instead it sets browser-like cipher suite and
// curve preferences that bring Go's ClientHello closer to real browsers —
// sufficient for most URL checking scenarios where servers only do basic
// User-Agent + TLS inspection.
package impersonate

import (
	"crypto/tls"
	"net/http"
)

const (
	profileNameChrome  = "chrome"
	profileNameFirefox = "firefox"
	profileNameSafari  = "safari"
)

// Profile defines a browser impersonation profile with TLS cipher/curve
// preferences and default HTTP headers matching the target browser.
type Profile struct {
	// Name is a human-readable identifier, e.g. "chrome", "firefox", "safari".
	Name string

	// CipherSuites is the ordered list of TLS 1.2 cipher suites (nil = use Go defaults).
	CipherSuites []uint16

	// CurvePreferences is the ordered list of ECDHE curves.
	CurvePreferences []tls.CurveID

	// DefaultHeaders are HTTP headers automatically added to every request,
	// mimicking the browser's default header set (User-Agent, Accept, Sec-CH-UA, etc.).
	DefaultHeaders http.Header
}

// chromeCiphers mirrors Chrome's TLS 1.3 preference (AES-GCM first) and
// adds TLS 1.2 fallback ciphers Chrome negotiates.
var chromeCiphers = []uint16{
	tls.TLS_AES_128_GCM_SHA256,
	tls.TLS_AES_256_GCM_SHA384,
	tls.TLS_CHACHA20_POLY1305_SHA256,
	tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
	tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
	tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
	tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
	tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
	tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
}

// chromeCurves mirrors Chrome's curve order: X25519 first, then NIST curves.
var chromeCurves = []tls.CurveID{
	tls.X25519,
	tls.CurveP256,
	tls.CurveP384,
}

// firefoxCurves mirrors Firefox's broader curve support.
var firefoxCurves = []tls.CurveID{
	tls.X25519,
	tls.CurveP256,
	tls.CurveP384,
	tls.CurveP521,
}

// ChromeProfile returns a profile with Chrome-like TLS preferences and headers.
func ChromeProfile() Profile {
	return Profile{
		Name:             profileNameChrome,
		CipherSuites:     chromeCiphers,
		CurvePreferences: chromeCurves,
		DefaultHeaders: http.Header{
			"User-Agent": {
				"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
			},
			"Accept":                    {"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8"},
			"Accept-Language":           {"en-US,en;q=0.9"},
			"Sec-Ch-Ua":                 {`"Google Chrome";v="131", "Chromium";v="131", "Not_A Brand";v="24"`},
			"Sec-Ch-Ua-Platform":        {`"Windows"`},
			"Sec-Ch-Ua-Mobile":          {"?0"},
			"Sec-Fetch-Site":            {"none"},
			"Sec-Fetch-Mode":            {"navigate"},
			"Sec-Fetch-User":            {"?1"},
			"Sec-Fetch-Dest":            {"document"},
			"Upgrade-Insecure-Requests": {"1"},
		},
	}
}

// FirefoxProfile returns a profile with Firefox-like TLS preferences and headers.
func FirefoxProfile() Profile {
	return Profile{
		Name:             profileNameFirefox,
		CipherSuites:     chromeCiphers, // Firefox uses same modern cipher suite order
		CurvePreferences: firefoxCurves,
		DefaultHeaders: http.Header{
			"User-Agent": {
				"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:133.0) Gecko/20100101 Firefox/133.0",
			},
			"Accept":                    {"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"},
			"Accept-Language":           {"en-US,en;q=0.5"},
			"Accept-Encoding":           {"gzip, deflate, br"},
			"Upgrade-Insecure-Requests": {"1"},
		},
	}
}

// SafariProfile returns a profile with Safari-like TLS preferences and headers.
func SafariProfile() Profile {
	return Profile{
		Name:             profileNameSafari,
		CipherSuites:     chromeCiphers, // Safari uses same modern cipher suite order
		CurvePreferences: chromeCurves,
		DefaultHeaders: http.Header{
			"User-Agent": {
				"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.2 Safari/605.1.15",
			},
			"Accept":          {"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"},
			"Accept-Language": {"en-US,en;q=0.9"},
			"Accept-Encoding": {"gzip, deflate, br"},
		},
	}
}

// ProfileByName returns the profile matching the given name, or ChromeProfile if unknown.
// Valid names: "chrome", "firefox", "safari".
func ProfileByName(name string) Profile {
	switch name {
	case profileNameFirefox:
		return FirefoxProfile()
	case profileNameSafari:
		return SafariProfile()
	default:
		return ChromeProfile()
	}
}
