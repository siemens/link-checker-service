// Copyright 2020-2023 Siemens AG
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log"
	"math/big"
	"net/http/httptest"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

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

// based on https://golang.org/src/crypto/tls/generate_cert.go
func createTestCertificates() (string, string, interface{}) {
	const key = "key.pem"
	const cert = "cert.pem"
	var err error
	var priv interface{}
	priv, _ = rsa.GenerateKey(rand.Reader, 2048)

	serial := new(big.Int)
	serial.SetInt64(1)
	template := x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{Organization: []string{"None"}},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(2 * time.Minute),
		KeyUsage:              x509.KeyUsageCRLSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		BasicConstraintsValid: true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, priv.(*rsa.PrivateKey).Public(), priv)
	if err != nil {
		log.Fatalf("Failed to create certificate: %v", err)
	}

	certOut, err := os.Create(cert)
	if err != nil {
		log.Fatalf("Failed to open %v for writing: %v", cert, err)
	}
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		log.Fatalf("Failed to write data to %v: %v", cert, err)
	}
	if err := certOut.Close(); err != nil {
		log.Fatalf("Error closing %v: %v", cert, err)
	}
	log.Printf("wrote %v", cert)

	keyOut, err := os.OpenFile(key, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Failed to open %v for writing: %v", key, err)
	}
	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		log.Fatalf("Unable to marshal private key: %v", err)
	}
	if err := pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privBytes}); err != nil {
		log.Fatalf("Failed to write data to %v: %v", key, err)
	}
	if err := keyOut.Close(); err != nil {
		log.Fatalf("Error closing %v: %v", key, err)
	}
	log.Printf("wrote %v", key)
	return cert, key, priv
}

func createJWTToken(privKey interface{}) (string, error) {
	token := jwt.New(jwt.SigningMethodRS384)
	token.Claims = &jwt.RegisteredClaims{
		ExpiresAt: &jwt.NumericDate{Time: time.Now().Add(3 * time.Minute).UTC()},
	}
	return token.SignedString(privKey)
}
