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
	"github.com/ethersphere/bee/pkg/bigint"
	"github.com/ethersphere/bee/pkg/crypto"
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

func (c *mockClient) Stamps(
	ctx context.Context,
) (client.StampsResponse, error) {
	var resp client.StampsResponse

	for batchID, st := range c.stamps {
		s := client.Stamp{
			Amount:        bigint.Wrap(st.amount),
			Depth:         st.depth,
			ImmutableFlag: st.immutable,
			BatchID:       batchID,
			Usable:        true,
		}

		resp.Stamps = append(resp.Stamps, s)
	}

	return resp, nil
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

	addr, err := c.upload(newCacAddress(data), data, batchID)

	resp := client.UploadResponse{Reference: addr}

	return resp, err
}

func (c *mockClient) upload(
	addresser addresser,
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

	addr, err := addresser()
	if err != nil {
		return swarm.ZeroAddress, fmt.Errorf("failed to create address: %w", err)
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

func (c *mockClient) DownloadChunk(
	ctx context.Context,
	addr swarm.Address,
) (io.ReadCloser, error) {
	return c.Download(ctx, addr)
}

func (c *mockClient) UploadSoc(
	ctx context.Context,
	owner common.Address,
	socID client.SocID,
	data []byte,
	signature client.SocSignature,
	batchID client.BatchID,
) (client.UploadSocResponse, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	addr, err := c.upload(newSocAddresser(owner, socID), makeSOCData(data), batchID)
	if err != nil {
		return client.UploadSocResponse{}, err
	}

	c.feeds[feedID(owner, hex.EncodeToString(socID))] = addr

	resp := client.UploadSocResponse{Reference: addr}

	return resp, nil
}

func (c *mockClient) FeedIndexLatest(
	ctx context.Context,
	owner common.Address,
	topic client.Topic,
) (client.FeedIndexResponse, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	var (
		addr   swarm.Address
		exists bool
		count  int
	)

	for i := uint64(0); ; i++ {
		socID, err := client.FeedID(topic, i)
		if err != nil {
			return client.FeedIndexResponse{}, fmt.Errorf("failed to generate feed id: %w", err)
		}

		addr, exists = c.feeds[feedID(owner, hex.EncodeToString(socID))]
		if !exists {
			break
		}

		count++
	}

	if count == 0 {
		return client.FeedIndexResponse{}, nil
	}

	resp := client.FeedIndexResponse{
		Reference: addr,
		Current:   uint64(count) - 1,
		Next:      uint64(count),
	}

	return resp, nil
}

func makeSOCData(data []byte) []byte {
	headerLen := swarm.HashSize + swarm.SocSignatureSize
	soc := make([]byte, headerLen, headerLen+len(data))

	return append(soc, data...)
}

func feedID(owner common.Address, id string) string {
	return fmt.Sprintf("%s-%s", owner.String(), id)
}

type addresser = func() (swarm.Address, error)

func newCacAddress(data []byte) addresser {
	//nolint:wrapcheck // relax
	return func() (swarm.Address, error) {
		hash, err := crypto.LegacyKeccak256(data)
		if err != nil {
			return swarm.ZeroAddress, err
		}

		return swarm.NewAddress(hash), nil
	}
}

func newSocAddresser(owner common.Address, socID client.SocID) addresser {
	//nolint:wrapcheck // relax
	return func() (swarm.Address, error) {
		ownerBytes := owner.Bytes()

		ref := make([]byte, 0, len(socID)+len(ownerBytes))
		ref = append(ref, socID...)
		ref = append(ref, ownerBytes...)

		hash, err := crypto.LegacyKeccak256(ref)
		if err != nil {
			return swarm.ZeroAddress, err
		}

		return swarm.NewAddress(hash), nil
	}
}

type dataReadCloser struct {
	io.Reader
}

func (rc *dataReadCloser) Close() error { return nil }
