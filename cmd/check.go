// Copyright 2020-2021 Siemens AG
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
// SPDX-License-Identifier: MPL-2.0
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/siemens/link-checker-service/infrastructure"
	"github.com/spf13/cobra"
)

// checkCmd represents the check command
var checkCmd = &cobra.Command{
	Use:   "check [url to check]",
	Short: "Checks a single URL without optimizations. Returns raw",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// disable logging so that there's only JSON output in the console
		log.SetOutput(ioutil.Discard)
		checker := infrastructure.NewURLCheckerClient()
		checkResult := checker.CheckURL(context.Background(), args[0])

		// prints a JSON-formatted raw check result representation
		b, err := json.MarshalIndent(checkResult, "", " ")
		if err != nil {
			log.Fatal(fmt.Errorf("ERROR: %v", err))
		}
		fmt.Println(string(b))
	},
}

func init() {
	rootCmd.AddCommand(checkCmd)
}
