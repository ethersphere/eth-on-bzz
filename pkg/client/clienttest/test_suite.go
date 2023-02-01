// Copyright 2023 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package clienttest

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"io"
	"math/big"
	"testing"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/ethersphere/eth-on-bzz/pkg/client"
	"github.com/ethersphere/eth-on-bzz/pkg/postage"
)

type TestSuite struct {
	suite.Suite
	ClientFact  func() client.Client
	PostageFact func(client.Client) postage.Postage
	PrivKey     *ecdsa.PrivateKey
}

func (suite *TestSuite) TestBuyStampOk() {
	t := suite.T()
	t.Parallel()

	c := suite.ClientFact()
	ctx := context.Background()

	stamp, err := c.BuyStamp(ctx, big.NewInt(10000000), 17, true)
	assert.NoError(t, err)
	assert.NotEmpty(t, stamp.BatchID)
}

func (suite *TestSuite) TestBuyStampError() {
	t := suite.T()
	t.Parallel()

	c := suite.ClientFact()
	ctx := context.Background()

	// Assert invalid depth
	stamp, err := c.BuyStamp(ctx, big.NewInt(10000000), 14, true)
	assert.Error(t, err)
	assert.Empty(t, stamp)

	// Assert low amount
	stamp, err = c.BuyStamp(ctx, big.NewInt(0), 16, true)
	assert.Error(t, err)
	assert.Empty(t, stamp)
}

func (suite *TestSuite) TestUploadDownloadOk() {
	t := suite.T()
	t.Parallel()

	c := suite.ClientFact()
	p := suite.PostageFact(c)
	ctx := context.Background()

	batchID, err := p.CurrentBatchID(ctx)
	assert.NoError(t, err)
	assert.NotEmpty(t, batchID)

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

		resp, err := c.Upload(ctx, data, batchID)
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

	c := suite.ClientFact()
	ctx := context.Background()

	data := randomBytes(t, 4)

	resp, err := c.Upload(ctx, data, client.BatchID("invalid"))
	assert.Error(t, err)
	assert.Empty(t, resp)
}

func (suite *TestSuite) TestDownloadError() {
	t := suite.T()
	t.Parallel()

	c := suite.ClientFact()
	ctx := context.Background()

	addr, err := client.RandomAddress()
	assert.NoError(t, err)

	reader, err := c.Download(ctx, addr)
	assert.Error(t, err)
	assert.Nil(t, reader)
}

func (suite *TestSuite) TestSocUploadOk() {
	t := suite.T()
	t.Parallel()

	c := suite.ClientFact()
	p := suite.PostageFact(c)
	ctx := context.Background()

	batchID, err := p.CurrentBatchID(ctx)
	assert.NoError(t, err)
	assert.NotEmpty(t, batchID)

	idRaw := randomBytes(t, swarm.HashSize)
	dataRaw := []byte("Ethereum blockchain data on Swarm")
	data, sig, owner, err := client.SignSocData(idRaw, dataRaw, suite.PrivKey)
	assert.NoError(t, err)

	resp, err := c.UploadSOC(ctx, owner, client.SocID(idRaw), data, sig, batchID)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)

	respReader, err := c.DownloadChunk(ctx, resp.Reference)
	assert.NoError(t, err)

	respData, err := io.ReadAll(respReader)
	assert.NoError(t, err)
	respReader.Close()

	assert.Equal(t, dataRaw, client.RawDataFromSOCResp(respData))
}

func randomBytes(t *testing.T, size int) []byte {
	t.Helper()

	buf := make([]byte, size)
	n, err := rand.Read(buf)
	assert.NoError(t, err)
	assert.Equal(t, size, n)

	return buf
}
