// Copyright 2020-2024 Siemens AG
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
// SPDX-License-Identifier: MPL-2.0

package server

import (
	"net/url"
	"strings"
)

func normalizedURL(u string) string {
	// just return the url for now
	// alternatives: 3rd party tool
	u = strings.TrimSpace(u)
	up, err := url.Parse(u)
	if err != nil {
		return u
	}
	return up.String()
}
