// Copyright 2020 Siemens AG
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
// SPDX-License-Identifier: MPL-2.0
package main

import "net/http/httptest"

// adapted the testing technique from
// https://github.com/gin-gonic/gin/blob/ce20f107f5dc498ec7489d7739541a25dcd48463/context_test.go#L1747-L1765 (MIT license)
// other techniques would not suit the needs due to the current Go library interfaces

// ResponseRecorder wrapper work-around to avoid missingMethod=CloseNotify TypeAssertionError
type closeNotifyRecorder struct {
	*httptest.ResponseRecorder
	closed chan bool
}

func (s *closeNotifyRecorder) CloseNotify() <-chan bool {
	return s.closed
}

func (s *closeNotifyRecorder) close() {
	s.closed <- true
}

func newCloseNotifyRecorder() *closeNotifyRecorder {
	return &closeNotifyRecorder{
		httptest.NewRecorder(),
		make(chan bool, 1),
	}
}
