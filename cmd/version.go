// Copyright 2020-2022 Siemens AG
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"fmt"

	"github.com/siemens/link-checker-service/infrastructure"
	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Prints the executable version",
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Println(infrastructure.BinaryVersion())
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
