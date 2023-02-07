// Copyright 2023 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bzzdb_test

import (
	"testing"

	"github.com/ethersphere/bee/pkg/crypto"
	"github.com/stretchr/testify/assert"

	"github.com/ethersphere/eth-on-bzz/pkg/bzzdb"
	"github.com/ethersphere/eth-on-bzz/pkg/bzzdb/dbtest"
	"github.com/ethersphere/eth-on-bzz/pkg/client/mock"
	"github.com/ethersphere/eth-on-bzz/pkg/postage"
)

func TestBzzDB(t *testing.T) {
	t.Parallel()

	privateKey, err := crypto.GenerateSecp256k1Key()
	assert.NoError(t, err)

	beeCli := mock.NewClient()

	newBzzDB := func() bzzdb.KeyValueStore {
		db, err := bzzdb.New(privateKey, beeCli, postage.New(beeCli))
		assert.NoError(t, err)

		return db
	}

	dbtest.TestDatabaseSuite(t, newBzzDB)
}
