// Copyright 2020-2024 Siemens AG
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
// SPDX-License-Identifier: MPL-2.0

package infrastructure

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
	"strings"
	"time"
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

func levelFormatter() func(i interface{}) string {
	return func(i interface{}) string {
		var l string
		if ll, ok := i.(string); ok {
			switch ll {
			case zerolog.LevelTraceValue:
				l = "TRACE"
			case zerolog.LevelDebugValue:
				l = "DEBUG"
			case zerolog.LevelInfoValue:
				l = "INFO "
			case zerolog.LevelWarnValue:
				l = "WARN "
			case zerolog.LevelErrorValue:
				l = "ERROR"
			case zerolog.LevelFatalValue:
				l = "FATAL"
			case zerolog.LevelPanicValue:
				l = "PANIC"
			default:
				l = "???  "
			}
		} else {
			if i == nil {
				l = "???  "
			} else {
				l = strings.ToUpper(fmt.Sprintf("%s     ", i))[0:5]
			}
		}
		return l
	}
}
