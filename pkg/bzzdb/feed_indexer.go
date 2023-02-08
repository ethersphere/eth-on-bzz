// Copyright 2023 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bzzdb

import (
	"context"
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethersphere/eth-on-bzz/pkg/client"
)

type FeedIndexFetcher interface {
	FeedIndexLatest(
		ctx context.Context,
		owner common.Address,
		topic client.Topic,
	) (client.FeedIndexResponse, error)
}

type Index = uint64

type FeedIndexer struct {
	indexFetcher FeedIndexFetcher
	owner        common.Address
	indexMap     map[string]*feedIndexData
	lock         sync.Mutex
}

type feedIndexData struct {
	current *Index
	next    Index
}

func NewFeedIndexer(indexFetcher FeedIndexFetcher, owner common.Address) *FeedIndexer {
	return &FeedIndexer{
		indexFetcher: indexFetcher,
		owner:        owner,
		indexMap:     make(map[string]*feedIndexData),
	}
}

func (i *FeedIndexer) AcquireNext(ctx context.Context, topic client.Topic) (Index, error) {
	key := hex.EncodeToString(topic)

	i.lock.Lock()
	defer i.lock.Unlock()

	indexData, ok := i.indexMap[key]

	if !ok {
		i.lock.Unlock()

		feedIndexResp, err := i.indexFetcher.FeedIndexLatest(ctx, i.owner, topic)
		if err != nil {
			return 0, fmt.Errorf("failed getting latest feed index: %w", err)
		}

		i.lock.Lock()

		if _, ok := i.indexMap[key]; !ok {
			indexData = newFeedIndexData(feedIndexResp)
			i.indexMap[key] = indexData
		}
	}

	index := indexData.next
	indexData.next++

	return index, nil
}

func (i *FeedIndexer) Release(topic client.Topic, index Index) {
	key := hex.EncodeToString(topic)

	i.lock.Lock()
	defer i.lock.Unlock()

	indexData, ok := i.indexMap[key]
	if !ok {
		return
	}

	if indexData.current == nil || index > *indexData.current {
		indexData.current = &index
	}
}

func (i *FeedIndexer) Current(ctx context.Context, topic client.Topic) (Index, bool, error) {
	key := hex.EncodeToString(topic)

	i.lock.Lock()
	defer i.lock.Unlock()

	indexData, ok := i.indexMap[key]

	if !ok {
		i.lock.Unlock()

		feedIndexResp, err := i.indexFetcher.FeedIndexLatest(ctx, i.owner, topic)
		if err != nil {
			return 0, false, fmt.Errorf("failed getting latest feed index: %w", err)
		}

		i.lock.Lock()

		if _, ok := i.indexMap[key]; !ok {
			indexData = newFeedIndexData(feedIndexResp)
			i.indexMap[key] = indexData
		}
	}

	if indexData.current == nil {
		return 0, false, nil
	}

	return *indexData.current, true, nil
}

func newFeedIndexData(resp client.FeedIndexResponse) *feedIndexData {
	if resp.Current == 0 && resp.Next == 0 {
		return &feedIndexData{
			current: nil,
			next:    0,
		}
	}

	return &feedIndexData{
		current: &resp.Current,
		next:    resp.Next,
	}
}
