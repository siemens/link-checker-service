// Copyright 2020-2022 Siemens AG
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
// SPDX-License-Identifier: MPL-2.0

package infrastructure

import (
	"github.com/google/uuid"
)

var instanceId string = ""

func GetInstanceId() string {
	return instanceId
}

func init() {
	instanceId = uuid.New().String()
}
