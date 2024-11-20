// Copyright 2020-2024 Siemens AG
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
// SPDX-License-Identifier: MPL-2.0

package infrastructure

import (
	"net"
	"net/url"
	"strings"
)

const badUrlPlaceholder = "<bad url>"
const noDomainPlaceholder = "<no domain or protocol>"

// DomainOf returns either the domain name or a placeholder in case of a parse error
func DomainOf(input string) string {
	u, err := url.Parse(input)
	if err != nil {
		// bad urls will be handled later by the client
		return badUrlPlaceholder
	}
	if strings.TrimSpace(u.Host) == "" {
		return noDomainPlaceholder
	}
	host, _, err := net.SplitHostPort(u.Host)
	if err != nil || host == "" {
		return u.Host
	}
	return host
}
