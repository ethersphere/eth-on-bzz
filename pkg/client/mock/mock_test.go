// Copyright 2023 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mock_test

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethersphere/bee/pkg/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/ethersphere/eth-on-bzz/pkg/client"
	"github.com/ethersphere/eth-on-bzz/pkg/client/mock"
)

func Test_Mock_Client(t *testing.T) {
	t.Parallel()

	privKey, err := crypto.GenerateSecp256k1Key()
	assert.NoError(t, err)

	currentBatchID := func(c client.Client) (client.BatchID, error) {
		resp, err := c.BuyStamp(context.Background(), big.NewInt(10000000), 22, true)
		if err != nil {
			return client.BatchID(""), err
		}

		return resp.BatchID, nil
	}

	suite.Run(t, &client.TestSuite{
		Fact:           mock.NewClient,
		CurrentBatchID: currentBatchID,
		PrivKey:        privKey,
	})
}
