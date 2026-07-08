// Copyright 2020-2024 Siemens AG
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
// SPDX-License-Identifier: MPL-2.0

package impersonate

import (
	"crypto/tls"
	"net/http"
	"testing"
)

func TestNewTransport_ReturnsRoundTripper(t *testing.T) {
	profile := ChromeProfile()
	transport := NewTransport(profile, false)

	if transport == nil {
		t.Fatal("expected non-nil transport")
	}

	if !transport.ForceAttemptHTTP2 {
		t.Error("expected ForceAttemptHTTP2 to be true")
	}

	if transport.TLSClientConfig == nil {
		t.Fatal("expected TLSClientConfig to be set")
	}

	if transport.TLSClientConfig.MinVersion != tls.VersionTLS12 {
		t.Error("expected MinVersion to be TLS 1.2")
	}

	if len(transport.TLSClientConfig.CipherSuites) == 0 {
		t.Error("expected CipherSuites to be configured")
	}

	if len(transport.TLSClientConfig.CurvePreferences) == 0 {
		t.Error("expected CurvePreferences to be configured")
	}
}

func TestNewTransport_WithSkipVerify(t *testing.T) {
	profile := ChromeProfile()
	transport := NewTransport(profile, true)

	if transport == nil {
		t.Fatal("expected non-nil transport")
	}

	if !transport.TLSClientConfig.InsecureSkipVerify {
		t.Error("expected InsecureSkipVerify to be true")
	}
}

func TestNewTransport_AllProfiles(t *testing.T) {
	profiles := []Profile{
		ChromeProfile(),
		FirefoxProfile(),
		SafariProfile(),
	}

	for _, p := range profiles {
		t.Run(p.Name, func(t *testing.T) {
			transport := NewTransport(p, false)
			if transport == nil {
				t.Fatal("expected non-nil transport")
			}
			if transport.TLSClientConfig == nil {
				t.Fatal("expected TLSClientConfig to be set")
			}
		})
	}
}

func TestNewTransport_ImplementsRoundTripper(t *testing.T) {
	profile := ChromeProfile()
	transport := NewTransport(profile, false)
	var _ http.RoundTripper = transport
}
