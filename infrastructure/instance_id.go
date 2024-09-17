// Copyright 2020-2023 Siemens AG
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
// SPDX-License-Identifier: MPL-2.0

package infrastructure

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

var instanceId = ""
var runningSince int64 = 0

func GetInstanceId() string {
	return instanceId
}

func GetRunningSince() string {
	return fmt.Sprintf("%v", runningSince)
}

func init() {
	instanceId = uuid.New().String()
	runningSince = time.Now().Unix()
}
