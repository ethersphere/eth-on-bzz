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
	envPrivateKey  = "PRIVATE_KEY"
)

func Test_Client_Integration(t *testing.T) {
	t.Parallel()

	cfg := client.Config{
		NodeURL: getEnv(t, envNodeAddress),
	}

	suite.Run(t, &clienttest.TestSuite{
		ClientFact: func() client.Client {
			return client.NewClient(cfg)
		},
		PostageFact: postage.New,
		PrivateKey:  getPrivateKey(t),
	})
}

func getPrivateKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()

	keyRaw := getEnv(t, envPrivateKey)
	key, err := crypto.DecodeSecp256k1PrivateKey([]byte(keyRaw))
	assert.NoError(t, err)

	return key
}

func getEnv(t *testing.T, env string) string {
	val := os.Getenv(env)
	if val == "" {
		assert.FailNow(t, "env variable is not provided", "missing env: %v", env)
	}

	return val
}
