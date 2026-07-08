// Copyright 2020-2024 Siemens AG
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
// SPDX-License-Identifier: MPL-2.0

package infrastructure

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func SetUpGlobalLogger() {
	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:         os.Stderr,
		FormatLevel: levelFormatter(),
		TimeFormat:  time.RFC3339,
	})
}

func SetUpConsoleLogging() {
	if os.Getenv("GIN_DISABLE_CONSOLE_COLOR") != "" {
		gin.DisableConsoleColor()
	}
}

var zerlogLevelLabels = map[string]string{
	zerolog.LevelTraceValue: "TRACE",
	zerolog.LevelDebugValue: "DEBUG",
	zerolog.LevelInfoValue:  "INFO ",
	zerolog.LevelWarnValue:  "WARN ",
	zerolog.LevelErrorValue: "ERROR",
	zerolog.LevelFatalValue: "FATAL",
	zerolog.LevelPanicValue: "PANIC",
}

func levelFormatter() func(i interface{}) string {
	return func(i interface{}) string {
		if ll, ok := i.(string); ok {
			if label, found := zerlogLevelLabels[ll]; found {
				return label
			}
			return "???  "
		}
		if i == nil {
			return "???  "
		}
		return strings.ToUpper(fmt.Sprintf("%s     ", i))[0:5]
	}
}
