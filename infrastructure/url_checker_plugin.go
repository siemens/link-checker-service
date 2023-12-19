// Copyright 2020-2023 Siemens AG
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
// SPDX-License-Identifier: MPL-2.0

package infrastructure

import "context"

// URLCheckerPlugin represents one low-level URL checker in a chain of checkers
type URLCheckerPlugin interface {
	// Name returns the name of the plugin to use in logging and result reporting
	Name() string

	// CheckURL gets the urlToCheck and lastResult, which it can process, and return the next result
	// and a boolean flag, whether the chain should be interrupted, and the last result - simply returned
	// ctx can be used to cancel the request prematurely
	CheckURL(ctx context.Context, urlToCheck string, lastResult *URLCheckResult) (*URLCheckResult, bool)
}
