// Copyright 2023 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mock

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethersphere/bee/pkg/postage/testing"
	"github.com/ethersphere/bee/pkg/swarm"

	"github.com/ethersphere/eth-on-bzz/pkg/client"
)

var (
	errInvalidStamp          = fmt.Errorf("invalid stamp")
	errStampUsageExceeded    = fmt.Errorf("stamp usage exceeded")
	errBuyStampInvalidAmount = fmt.Errorf("amount must be positive non zero value")
	errBuyStampInvalidDepth  = fmt.Errorf("depth is not in acceptable range")
)

func NewClient() client.Client {
	return &mockClient{
		stamps: make(map[client.BatchID]*stampData),
		data:   make(map[string][]byte),
		feeds:  make(map[string]swarm.Address),
	}
}

type mockClient struct {
	stamps map[client.BatchID]*stampData
	data   map[string][]byte
	feeds  map[string]swarm.Address
	lock   sync.Mutex
}

type stampData struct {
	amount    *big.Int
	depth     uint8
	immutable bool
	usage     int
}

const (
	chunkSize   = 4096
	bucketDepth = 16
	minDepth    = bucketDepth + 1
	maxDepth    = 255
)

func (s *stampData) incUsage(size int) error {
	requiredChunks := int(math.Ceil(float64(size) / chunkSize))
	maxChunks := 1 << (s.depth - bucketDepth)

	if s.usage+requiredChunks > maxChunks {
		return errStampUsageExceeded
	}

	s.usage += requiredChunks

	return nil
}

func (c *mockClient) BuyStamp(
	ctx context.Context,
	amount *big.Int,
	depth uint8,
	immutable bool,
) (client.BuyStampResponse, error) {
	if amount.Cmp(big.NewInt(0)) <= 0 {
		return client.BuyStampResponse{}, errBuyStampInvalidAmount
	}

	if depth < minDepth || depth > maxDepth {
		return client.BuyStampResponse{}, errBuyStampInvalidDepth
	}

	batchID := client.BatchID(hex.EncodeToString(testing.MustNewID()))

	c.lock.Lock()
	c.stamps[batchID] = &stampData{
		amount:    amount,
		depth:     depth,
		immutable: immutable,
	}
	c.lock.Unlock()

	resp := client.BuyStampResponse{
		BatchID: batchID,
	}

	return resp, nil
}

func (c *mockClient) Upload(
	ctx context.Context,
	data []byte,
	batchID client.BatchID,
) (client.UploadResponse, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	addr, err := c.upload(ctx, data, batchID)

	resp := client.UploadResponse{Reference: addr}

	return resp, err
}

func (c *mockClient) upload(
	ctx context.Context,
	data []byte,
	batchID client.BatchID,
) (swarm.Address, error) {
	stamp, exists := c.stamps[batchID]
	if !exists {
		return swarm.ZeroAddress, errInvalidStamp
	}

	if err := stamp.incUsage(len(data)); err != nil {
		return swarm.ZeroAddress, fmt.Errorf("stamp usage exceeded: %w", err)
	}

	addr, err := c.newUniqueAddress(ctx)
	if err != nil {
		return swarm.ZeroAddress, fmt.Errorf("failed to create unique address: %w", err)
	}

	c.data[addr.ByteString()] = data
	c.stamps[batchID] = stamp

	return addr, nil
}

func (c *mockClient) Download(
	ctx context.Context,
	addr swarm.Address,
) (io.ReadCloser, error) {
	c.lock.Lock()
	data, exists := c.data[addr.ByteString()]
	c.lock.Unlock()

	if !exists {
		return nil, client.ErrNotFound
	}

	rc := &dataReadCloser{
		Reader: bytes.NewReader(data),
	}

	return rc, nil
}

func (c *mockClient) UploadSOC(
	ctx context.Context,
	owner common.Address,
	id client.SocID,
	data []byte,
	signature client.SocSignature,
	batchID client.BatchID,
) (client.UploadSOCResponse, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	addr, err := c.upload(ctx, data, batchID)
	if err != nil {
		return client.UploadSOCResponse{}, err
	}

	c.feeds[feedID(owner, string(id))] = addr

	resp := client.UploadSOCResponse{Reference: addr}

	return resp, nil
}

func (c *mockClient) FeedGet(
	ctx context.Context,
	owner common.Address,
	topic client.Topic,
) (client.FeedGetResponse, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	addr, exists := c.feeds[feedID(owner, string(topic))]
	if !exists {
		return client.FeedGetResponse{}, client.ErrNotFound
	}

	data, exists := c.data[addr.ByteString()]
	if !exists {
		return client.FeedGetResponse{}, client.ErrNotFound
	}

	resp := client.FeedGetResponse{
		Reference: addr,
		Current:   data,
	}

	return resp, nil
}

func feedID(owner common.Address, id string) string {
	return fmt.Sprintf("%s-%s", owner.String(), id)
}

func (c *mockClient) newUniqueAddress(ctx context.Context) (swarm.Address, error) {
	for {
		select {
		case <-ctx.Done():
			return swarm.ZeroAddress, ctx.Err() //nolint:wrapcheck // relax

		default:
			addr, err := client.RandomAddress()
			if err != nil {
				continue
			}

			if _, exists := c.data[addr.ByteString()]; !exists {
				return addr, nil
			}
		}
	}
}

type dataReadCloser struct {
	io.Reader
}

func (rc *dataReadCloser) Close() error { return nil }
