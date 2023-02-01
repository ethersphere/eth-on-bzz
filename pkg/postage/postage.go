// Copyright 2023 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package postage

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"

	"github.com/ethersphere/eth-on-bzz/pkg/client"
)

type Postage interface {
	CurrentBatchID(context.Context) (client.BatchID, error)
}

func New(beeCli client.Client) Postage {
	return &postage{
		beeCli: beeCli,
	}
}

type postage struct {
	beeCli  client.Client
	batchID client.BatchID
	lock    sync.Mutex
}

func (p *postage) CurrentBatchID(ctx context.Context) (client.BatchID, error) {
	p.lock.Lock()
	batchID := p.batchID
	p.lock.Unlock()

	if batchID != "" {
		return batchID, nil
	}

	batchID, err := p.fetchOrBuyStamp(ctx)
	if err != nil {
		return batchID, err
	}

	p.lock.Lock()
	p.batchID = batchID
	p.lock.Unlock()

	return batchID, nil
}

func (p *postage) fetchOrBuyStamp(ctx context.Context) (client.BatchID, error) {
	batchID, err := p.fetchFirstUsableStamp(ctx)
	if errors.Is(err, errNoUsableBatch) {
		resp, err := p.beeCli.BuyStamp(ctx, big.NewInt(10000000), 22, true)
		if err != nil {
			return client.BatchID(""), err
		}

		return resp.BatchID, nil
	}

	return batchID, err
}

var errNoUsableBatch = fmt.Errorf("no usable batch found")

func (p *postage) fetchFirstUsableStamp(ctx context.Context) (client.BatchID, error) {
	resp, err := p.beeCli.Stamps(ctx)
	if err != nil {
		return client.BatchID(""), err
	}

	for _, st := range resp.Stamps {
		if st.Usable && st.Exists {
			return st.BatchID, nil
		}
	}

	return client.BatchID(""), errNoUsableBatch
}
