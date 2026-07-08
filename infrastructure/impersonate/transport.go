// Copyright 2020-2024 Siemens AG
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
// SPDX-License-Identifier: MPL-2.0

package impersonate

import (
	"crypto/tls"
	"net/http"
)

// NewTransport creates an http.Transport with browser-like TLS configuration
// using only the standard library crypto/tls. It sets the profile's cipher suite
// and curve preferences to bring Go's ClientHello closer to the target browser.
//
// Exact TLS fingerprint matching is not achievable with stock crypto/tls (that
// would require a fork like utls for extension ordering, GREASE, ALPS, etc.).
// However, for many URL checking scenarios, browser-appropriate cipher/curve
// preferences plus the right HTTP headers are sufficient.
//
// The returned transport is compatible with net/http.Client and resty.Client
// via SetTransport(). HTTP/2 is enabled via ForceAttemptHTTP2.
func NewTransport(profile Profile, skipVerify bool) *http.Transport {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: skipVerify,
		MinVersion:         tls.VersionTLS12,
	}

	if len(profile.CipherSuites) > 0 {
		tlsConfig.CipherSuites = profile.CipherSuites
	}
	if len(profile.CurvePreferences) > 0 {
		tlsConfig.CurvePreferences = profile.CurvePreferences
	}

	return &http.Transport{
		TLSClientConfig:   tlsConfig,
		ForceAttemptHTTP2: true,
	}
}

