// Copyright 2023 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethersphere/bee/pkg/cac"
	"github.com/ethersphere/bee/pkg/crypto"
	"github.com/ethersphere/bee/pkg/soc"
	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type TestSuite struct {
	suite.Suite
	Fact func() Client
}

func (suite *TestSuite) TestStamps() {
	t := suite.T()
	t.Parallel()

	c := suite.Fact()
	ctx := context.Background()

	stamp, err := c.BuyStamp(ctx, big.NewInt(10000000), 17, true)
	assert.NoError(t, err)
	assert.NotEmpty(t, stamp.BatchID)

	stamp, err = c.BuyStamp(ctx, big.NewInt(10000000), 16, true)
	assert.Error(t, err)
	assert.Empty(t, stamp)

	stamp, err = c.BuyStamp(ctx, big.NewInt(0), 16, true)
	assert.Error(t, err)
	assert.Empty(t, stamp)
}

func (suite *TestSuite) TestUploadDownload() {
	t := suite.T()
	t.Parallel()

	c := suite.Fact()
	ctx := context.Background()

	stamp, err := c.BuyStamp(ctx, big.NewInt(10000000), 22, true)
	assert.NoError(t, err)
	assert.NotEmpty(t, stamp.BatchID)

	tests := []struct {
		size int
	}{
		{size: 0},
		{size: 1},
		{size: 4},
		{size: 1024},
		{size: 4096},
		{size: 8192},
		{size: 16384},
	}

	for _, tc := range tests {
		data := randomBytes(t, tc.size)

		resp, err := c.Upload(ctx, data, stamp.BatchID)
		assert.NoError(t, err)
		assert.NotEmpty(t, resp)

		reader, err := c.Download(ctx, resp.Reference)
		assert.NoError(t, err)
		assert.NotNil(t, reader)

		downloadedData, err := io.ReadAll(reader)
		assert.NoError(t, err)

		assert.Equal(t, data, downloadedData)
	}
}

func (suite *TestSuite) TestUploadError() {
	t := suite.T()
	t.Parallel()

	c := suite.Fact()
	ctx := context.Background()

	data := randomBytes(t, 4)

	resp, err := c.Upload(ctx, data, BatchID("invalid"))
	assert.Error(t, err)
	assert.Empty(t, resp)
}

func (suite *TestSuite) TestDownloadError() {
	t := suite.T()
	t.Parallel()

	c := suite.Fact()
	ctx := context.Background()

	addr, err := RandomAddress()
	assert.NoError(t, err)

	reader, err := c.Download(ctx, addr)
	assert.Error(t, err)
	assert.Nil(t, reader)
}

func (suite *TestSuite) TestSocUpload() {
	t := suite.T()
	t.Parallel()

	c := suite.Fact()
	ctx := context.Background()

	stamp, err := c.BuyStamp(ctx, big.NewInt(10000000), 22, true)
	assert.NoError(t, err)
	assert.NotEmpty(t, stamp.BatchID)

	id := []byte("ethswarm-key-1")
	data := []byte("Ethereum blockchain data on Swarm")

	sig, owner, err := prepareSocData(id, data)
	assert.NoError(t, err)

	resp, err := c.UploadSOC(ctx, owner, string(id), data, sig, stamp.BatchID)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func randomBytes(t *testing.T, size int) []byte {
	t.Helper()

	buf := make([]byte, size)
	n, err := rand.Read(buf)
	assert.NoError(t, err)
	assert.Equal(t, size, n)

	return buf
}

//nolint:wrapcheck //relax
func prepareSocData(id, payload []byte) (string, common.Address, error) {
	privKey, err := crypto.GenerateSecp256k1Key()
	if err != nil {
		return "", common.Address{}, err
	}

	signer := crypto.NewDefaultSigner(privKey)

	publicKey, err := signer.PublicKey()
	if err != nil {
		return "", common.Address{}, err
	}

	ch, err := cac.New(payload)
	if err != nil {
		return "", common.Address{}, err
	}

	sch, err := soc.New(id, ch).Sign(signer)
	if err != nil {
		return "", common.Address{}, err
	}

	chunkData := sch.Data()
	signatureBytes := chunkData[swarm.HashSize : swarm.HashSize+swarm.SocSignatureSize]
	signature := hex.EncodeToString(signatureBytes)

	ownerBytes, err := crypto.NewEthereumAddress(*publicKey)
	if err != nil {
		return "", common.Address{}, err
	}

	owner := common.BytesToAddress(ownerBytes)

	return signature, owner, nil
}
