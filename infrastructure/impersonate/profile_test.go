// Copyright 2020-2024 Siemens AG
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
// SPDX-License-Identifier: MPL-2.0

package impersonate

import (
	"testing"
)

func TestChromeProfile(t *testing.T) {
	p := ChromeProfile()

	if p.Name != "chrome" {
		t.Errorf("expected name 'chrome', got %q", p.Name)
	}

	if len(p.CipherSuites) == 0 {
		t.Error("expected CipherSuites to be set")
	}

	if len(p.CurvePreferences) == 0 {
		t.Error("expected CurvePreferences to be set")
	}

	if p.DefaultHeaders.Get("User-Agent") == "" {
		t.Error("expected User-Agent header to be set")
	}

	if p.DefaultHeaders.Get("Sec-Ch-Ua") == "" {
		t.Error("expected Sec-Ch-Ua header to be set")
	}

	if p.DefaultHeaders.Get("Accept") == "" {
		t.Error("expected Accept header to be set")
	}
}

func TestFirefoxProfile(t *testing.T) {
	p := FirefoxProfile()

	if p.Name != "firefox" {
		t.Errorf("expected name 'firefox', got %q", p.Name)
	}

	if len(p.CipherSuites) == 0 {
		t.Error("expected CipherSuites to be set")
	}

	if len(p.CurvePreferences) == 0 {
		t.Error("expected CurvePreferences to be set")
	}

	if p.DefaultHeaders.Get("User-Agent") == "" {
		t.Error("expected User-Agent header to be set")
	}
}

func TestSafariProfile(t *testing.T) {
	p := SafariProfile()

	if p.Name != "safari" {
		t.Errorf("expected name 'safari', got %q", p.Name)
	}

	if len(p.CipherSuites) == 0 {
		t.Error("expected CipherSuites to be set")
	}

	if len(p.CurvePreferences) == 0 {
		t.Error("expected CurvePreferences to be set")
	}

	if p.DefaultHeaders.Get("User-Agent") == "" {
		t.Error("expected User-Agent header to be set")
	}
}

func TestProfileByName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"chrome", "chrome", "chrome"},
		{"firefox", "firefox", "firefox"},
		{"safari", "safari", "safari"},
		{"unknown returns chrome", "edge", "chrome"},
		{"empty returns chrome", "", "chrome"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := ProfileByName(tt.input)
			if p.Name != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, p.Name)
			}
		})
	}
}
