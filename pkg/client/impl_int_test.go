// Copyright 2023 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build integration
// +build integration

package client_test

import (
	"crypto/ecdsa"
	"os"
	"testing"

	"github.com/ethersphere/bee/pkg/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/ethersphere/eth-on-bzz/pkg/client"
	"github.com/ethersphere/eth-on-bzz/pkg/client/clienttest"
	"github.com/ethersphere/eth-on-bzz/pkg/postage"
)

const (
	envNodeAddress = "NODE_ADDRESS"
	envPrivKey     = "PRIVATE_KEY"
)

func Test_Client_Integration(t *testing.T) {
	t.Parallel()

	// os.Setenv(envNodeAddress, "http://localhost")
	// os.Setenv(envPrivKey, "dc85109859ffd3a1256fda9f0570c28c")

	privKey := getPrivKey(t)

	cfg := client.Config{
		NodeURL: getEnv(t, envNodeAddress),
	}

	suite.Run(t, &clienttest.TestSuite{
		ClientFact: func() client.Client {
			return client.NewClient(cfg)
		},
		PostageFact: postage.New,
		PrivKey:     privKey,
	})
}

func getPrivKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()

	privKeyRaw := getEnv(t, envPrivKey)
	privKey, err := crypto.DecodeSecp256k1PrivateKey([]byte(privKeyRaw))
	assert.NoError(t, err)

	return privKey
}

func getEnv(t *testing.T, env string) string {
	val := os.Getenv(env)
	if val == "" {
		assert.FailNow(t, "env variable is not provided", "missing env: %v", env)
	}

	return val
}
