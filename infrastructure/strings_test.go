// Copyright 2020-2021 Siemens AG
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
// SPDX-License-Identifier: MPL-2.0
package infrastructure

import (
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestSanitizingUserLogInputForNewlines(t *testing.T) {
	input := "bad\n\r\ninput"
	output := sanitizeUserLogInput(input)
	assert.NotContains(t, output, "\n")
	assert.NotContains(t, output, "\r")
}

func TestSanitizingUserLogInputOnMaxLength(t *testing.T) {
	input := "\n\r\n" /*3*/ + strings.Repeat("x", 101)
	assert.Equal(t, len(input), 101+3, "test sanity check")

	output := sanitizeUserLogInput(input)
	assert.LessOrEqual(t, len(output), 100)
	assert.NotContains(t, output, "\r")
}
