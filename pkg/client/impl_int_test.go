// Copyright 2023 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build integration
// +build integration

package client_test

import (
	"os"
	"testing"

	"github.com/ethersphere/bee/pkg/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/ethersphere/eth-on-bzz/pkg/client"
)

func Test_Client_Integration(t *testing.T) {
	t.Parallel()

	// os.Setenv("NODE_ADDRESS", "http://localhost:1633")

	privKey, err := crypto.GenerateSecp256k1Key()
	assert.NoError(t, err)

	cfg := client.Config{
		NodeURL: getNodeURL(t),
	}

	suite.Run(t, &client.TestSuite{
		Fact: func() client.Client {
			return client.NewClient(cfg)
		},
		CurrentBatchID: func(c client.Client) (client.BatchID, error) {
			return client.BatchID("93f9ac6dd0bd81a4c3111437caf174c3c33e8f4488781e88a9bc746cf83d47da"), nil
		},
		PrivKey: privKey,
	})
}

func getNodeURL(t *testing.T) string {
	t.Helper()

	nodeURL := os.Getenv("NODE_ADDRESS")
	if nodeURL == "" {
		t.Fatal("bee node address is not provided")
	}

	return nodeURL
}
