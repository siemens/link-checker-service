// Copyright 2020-2022 Siemens AG
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
// SPDX-License-Identifier: MPL-2.0

package infrastructure

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDomainOf(t *testing.T) {
	assert.Equal(t, badUrlPlaceholder, DomainOf("123://bad"))
	assert.Equal(t, noDomainPlaceholder, DomainOf(" "))
	assert.Equal(t, "example.com", DomainOf("https://example.com/123"))
	assert.Equal(
		t,
		noDomainPlaceholder,
		DomainOf("example.com/123"),
		"urls are expected to be prefixed at an earlier processing stage",
	)
	assert.Equal(t, "example.com", DomainOf("https://example.com:42/placeholder/"))
	assert.Equal(t, "localhost", DomainOf("ftp://localhost/123"))
	assert.Equal(t, "127.0.0.1", DomainOf("https://127.0.0.1:8080/123"))
}
