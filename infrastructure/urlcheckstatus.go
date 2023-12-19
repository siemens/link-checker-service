// Copyright 2020-2023 Siemens AG
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
// SPDX-License-Identifier: MPL-2.0
//go:generate go install github.com/alvaroloes/enumer@master
//go:generate enumer -type=URLCheckStatus -json -text -transform=lower

package infrastructure

//run go generate to generate enum serialization code

// URLCheckStatus indicates the URL check outcome
type URLCheckStatus int

const (
	// Skipped indicates that the URL check wasn't performed
	Skipped URLCheckStatus = iota
	// Ok indicates the URL is accessible
	Ok
	// Broken indicates the URL cannot be accessed for some reason
	Broken
	// Dropped indicates an internal reason for not proceeding with the URL check
	Dropped
)
