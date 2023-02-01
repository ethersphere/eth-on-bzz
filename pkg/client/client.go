// Copyright 2023 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"context"
	"fmt"
	"io"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethersphere/bee/pkg/bigint"
	"github.com/ethersphere/bee/pkg/soc"
	"github.com/ethersphere/bee/pkg/swarm"
)

var ErrNotFound = fmt.Errorf("not found")

type (
	BatchID string // hex encoded [32]byte

	SocID = soc.ID

	SocSignature string // hex encoded [65]bytes (swarm.SocSignatureSize)

	Topic string // hex encoded []byte

	UploadResponse struct {
		Reference swarm.Address `json:"reference"`
	}

	StampsResponse struct {
		Stamps []Stamp `json:"stamps"`
	}

	Stamp struct {
		BatchID       BatchID        `json:"batchID"`
		Utilization   uint32         `json:"utilization"`
		Usable        bool           `json:"usable"`
		Label         string         `json:"label"`
		Depth         uint8          `json:"depth"`
		Amount        *bigint.BigInt `json:"amount"`
		BucketDepth   uint8          `json:"bucketDepth"`
		BlockNumber   uint64         `json:"blockNumber"`
		ImmutableFlag bool           `json:"immutableFlag"`
		Exists        bool           `json:"exists"`
		BatchTTL      int64          `json:"batchTTL"`
		Expired       bool           `json:"expired"`
	}

	BuyStampResponse struct {
		BatchID BatchID `json:"batchID"`
	}

	UploadSOCResponse struct {
		Reference swarm.Address `json:"reference"`
	}

	FeedGetResponse struct {
		Reference swarm.Address `json:"reference"`
		Current   []byte        // SwarmFeedIndexHeader
	}

	// Client is interface for communicating with Bee node API.
	Client interface {
		// Stamps fetches purchased stamp batches via /stamps endpoint.
		Stamps(
			ctx context.Context,
		) (StampsResponse, error)

		// BuyStamp buys a new postage stamp batch.
		BuyStamp(
			ctx context.Context,
			amount *big.Int,
			depth uint8,
			immutable bool,
		) (BuyStampResponse, error)

		// Upload arbitrary bytes data via /bytes endpoint.
		Upload(
			ctx context.Context,
			data []byte,
			batchID BatchID,
		) (UploadResponse, error)

		// Download bytes data via /bytes endpoint.
		Download(
			ctx context.Context,
			addr swarm.Address,
		) (io.ReadCloser, error)

		// DownloadChunk downloads Chunk via /chunks endpoint.
		DownloadChunk(
			ctx context.Context,
			addr swarm.Address,
		) (io.ReadCloser, error)

		// UploadSOC uploads Single Owner Chunk data via /soc endpoint.
		UploadSOC(
			ctx context.Context,
			owner common.Address,
			id SocID,
			data []byte,
			signature SocSignature,
			batchID BatchID,
		) (UploadSOCResponse, error)

		// FeedGet returns the most recent feed data from /feed/owner/topic.
		FeedGet(
			ctx context.Context,
			owner common.Address,
			topic Topic,
		) (FeedGetResponse, error)
	}
)
