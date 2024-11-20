// Copyright 2020-2024 Siemens AG
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
// SPDX-License-Identifier: MPL-2.0

package infrastructure

import "runtime/debug"

// Version is a global variable written by the linker during CI builds
var Version string

// BinaryVersion returns the best guess at the server's version
func BinaryVersion() string {
	if Version != "" {
		return Version
	}
	version := "unknown"
	info, ok := debug.ReadBuildInfo()
	if ok && info.Main.Version != "" {
		version = info.Main.Version
	}
	return version
}
