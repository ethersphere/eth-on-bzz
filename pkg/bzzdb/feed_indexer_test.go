// Copyright 2023 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bzzdb_test

import (
	"context"
	"crypto/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"

	"github.com/ethersphere/eth-on-bzz/pkg/bzzdb"
	"github.com/ethersphere/eth-on-bzz/pkg/client"
)

func Test_FeedIndexer(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	indexer := bzzdb.NewFeedIndexer(&feedIndexFetcher{}, common.Address{})

	topic := makeTopic(t)
	for i := 0; i < 10; i++ {
		index, err := indexer.AcquireNext(ctx, topic)
		assert.NoError(t, err)
		assert.Equal(t, bzzdb.Index(i), index)

		current, exists, err := indexer.Current(ctx, topic)
		assert.NoError(t, err)
		assert.False(t, exists)
		assert.Equal(t, bzzdb.Index(0), current)
	}

	for i := 0; i < 10; i++ {
		indexer.Release(topic, bzzdb.Index(i))

		current, exists, err := indexer.Current(ctx, topic)
		assert.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, bzzdb.Index(i), current)
	}
}

type feedIndexFetcher struct{}

func (i *feedIndexFetcher) FeedIndexLatest(
	ctx context.Context,
	owner common.Address,
	topic client.Topic,
) (client.FeedIndexResponse, error) {
	return client.FeedIndexResponse{
		Current: 0,
		Next:    0,
	}, nil
}

func makeTopic(t *testing.T) client.Topic {
	t.Helper()

	size := 32

	buf := make([]byte, size)
	n, err := rand.Read(buf)
	assert.NoError(t, err)
	assert.Equal(t, size, n)

	return buf
}
