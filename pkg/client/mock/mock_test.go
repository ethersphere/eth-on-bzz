// Copyright 2023 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mock_test

import (
	"testing"

	"github.com/ethersphere/bee/pkg/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/ethersphere/eth-on-bzz/pkg/client/clienttest"
	"github.com/ethersphere/eth-on-bzz/pkg/client/mock"
	"github.com/ethersphere/eth-on-bzz/pkg/postage"
)

func Test_Mock_Client(t *testing.T) {
	t.Parallel()

	privKey, err := crypto.GenerateSecp256k1Key()
	assert.NoError(t, err)

	suite.Run(t, &clienttest.TestSuite{
		ClientFact:  mock.NewClient,
		PostageFact: postage.New,
		PrivKey:     privKey,
	})
}
