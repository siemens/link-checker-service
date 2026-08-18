// Copyright 2020-2024 Siemens AG
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
// SPDX-License-Identifier: MPL-2.0

package server

import (
	"testing"

	ginGwt "github.com/appleboy/gin-jwt/v2"
	"github.com/stretchr/testify/assert"
)

func Test_tryGetJwksKeyFunc_returnsNilOnBadURL(t *testing.T) {
	kf := tryGetJwksKeyFunc("http://127.0.0.1:1/nonexistent")
	assert.Nil(t, kf, "unreachable JWKS URL should return nil KeyFunc")
}

func Test_nilKeyFuncWithInvalidKeyFiles_failsClosed(t *testing.T) {
	// Simulates what happens when JWKS fetch fails (KeyFunc=nil)
	// and key files are invalid: ginGwt.New must return an error.
	_, err := ginGwt.New(&ginGwt.GinJWTMiddleware{
		KeyFunc:          nil,
		PrivKeyFile:      "/nonexistent/priv.pem",
		PubKeyFile:       "/nonexistent/pub.pem",
		SigningAlgorithm: "RS384",
	})
	assert.Error(t, err, "nil KeyFunc with invalid key files must fail (fail-closed)")
}
