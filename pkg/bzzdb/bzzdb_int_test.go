// Copyright 2023 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build integration
// +build integration

package bzzdb_test

import (
	"crypto/ecdsa"
	"os"
	"testing"

	"github.com/ethersphere/bee/pkg/crypto"
	"github.com/stretchr/testify/assert"

	"github.com/ethersphere/eth-on-bzz/pkg/bzzdb"
	"github.com/ethersphere/eth-on-bzz/pkg/bzzdb/dbtest"
	"github.com/ethersphere/eth-on-bzz/pkg/client"
)

const (
	envNodeAddress = "NODE_ADDRESS"
	envPrivateKey  = "PRIVATE_KEY"
)

func Test_BzzDB_Integration(t *testing.T) {
	t.Parallel()

	privKey, err := crypto.GenerateSecp256k1Key()
	assert.NoError(t, err)

	beeCli := client.NewClient(client.Config{
		NodeURL: getEnv(t, envNodeAddress),
	})

	newBzzDB := func() bzzdb.KeyValueStore {
		db, err := bzzdb.New(privKey, beeCli)
		assert.NoError(t, err)

		return db
	}

	dbtest.TestDatabaseSuite(t, newBzzDB)
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
