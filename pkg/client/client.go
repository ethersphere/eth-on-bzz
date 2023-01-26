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
	"github.com/ethersphere/bee/pkg/swarm"
)

var ErrNotFound = fmt.Errorf("not found")

type (
	BatchID string // 32bytes hex encoded string

	SocSignature string // 65bytes (swarm.SocSignatureSize) hex encoded string

	UploadResponse struct {
		Reference swarm.Address `json:"reference"`
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

	// Client is interface for communicating with Bee node.
	Client interface {
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

		// UploadSOC uploads Single Owner Chunk data via /soc endpoint.
		UploadSOC(
			ctx context.Context,
			owner common.Address,
			id string,
			data []byte,
			signature SocSignature,
			batchID BatchID,
		) (UploadSOCResponse, error)

		// FeedGet returns the most recent feed data from /feed/owner/topic.
		FeedGet(
			ctx context.Context,
			owner common.Address,
			topic string,
		) (FeedGetResponse, error)
	}
)
