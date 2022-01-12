// Copyright 2020-2021 Siemens AG
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
// SPDX-License-Identifier: MPL-2.0
package infrastructure

import "strings"

const loggingUserDataMaxLength = 100

func sanitizeUserLogInput(input string) string {
	var res = input
	res = strings.ReplaceAll(res, "\n", " ")
	res = strings.ReplaceAll(res, "\r", " ")
	if len(res) > loggingUserDataMaxLength {
		res = res[:loggingUserDataMaxLength]
	}
	return res
}
