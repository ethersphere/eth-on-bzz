// Copyright 2023 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bzzdb

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"errors"
	"io"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethersphere/bee/pkg/crypto"
	"github.com/ethersphere/bee/pkg/swarm"

	"github.com/ethersphere/eth-on-bzz/pkg/client"
	"github.com/ethersphere/eth-on-bzz/pkg/postage"
)

//nolint:gochecknoglobals
var (
	errBzzDBNotFound = errors.New("not found")

	zeroSocData = make([]byte, swarm.HashSize)
	keyPrefix   = []byte("bzzdb-")
)

//nolint:wrapcheck //relax
func New(
	privateKey *ecdsa.PrivateKey,
	beeCli client.Client,
	postage postage.Postage,
) (KeyValueStore, error) {
	owner, err := client.OwnerFromKey(privateKey)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &bzzdb{
		privateKey: privateKey,
		owner:      owner,
		beeCli:     beeCli,
		postage:    postage,
		indexer:    NewFeedIndexer(beeCli, owner),
		ctx:        ctx,
		ctxCancel:  cancel,
	}, nil
}

// bzzdb implements ethereum KeyValueStore interface.
type bzzdb struct {
	privateKey *ecdsa.PrivateKey
	beeCli     client.Client
	postage    postage.Postage
	owner      common.Address
	indexer    *FeedIndexer

	//nolint:containedctx // this ctx is need because methods of KeyValueStore
	// interface do not pass down context. Single context is created in New method
	// and reused for all Bee Client calls.
	ctx       context.Context
	ctxCancel context.CancelFunc
}

func (db *bzzdb) Has(key []byte) (bool, error) {
	if _, err := db.Get(key); err != nil {
		if errors.Is(err, errBzzDBNotFound) ||
			errors.Is(err, client.ErrNotFound) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

//nolint:wrapcheck //relax
func (db *bzzdb) Get(key []byte) ([]byte, error) {
	topic, err := makeTopic(key)
	if err != nil {
		return nil, err
	}

	index, exists, err := db.indexer.Current(db.ctx, topic)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, errBzzDBNotFound
	}

	ref, err := client.FeedUpdateReference(db.owner, topic, index)
	if err != nil {
		return nil, err
	}

	respData, err := db.downloadAndRead(db.beeCli.DownloadChunk, swarm.NewAddress(ref))
	if err != nil {
		return nil, err
	}

	respData = client.PayloadStripTime(client.RawDataFromSocResp(respData))

	if bytes.Equal(respData, zeroSocData) {
		return nil, errBzzDBNotFound
	}

	respData, err = db.downloadAndRead(db.beeCli.DownloadBytes, swarm.NewAddress(respData))
	if err != nil {
		return nil, err
	}

	return respData, nil
}

//nolint:wrapcheck //relax
func (db *bzzdb) Put(key []byte, value []byte) error {
	uploadRespC := db.uploadAsync(value)

	batchID, err := db.postage.CurrentBatchID(db.ctx)
	if err != nil {
		return err
	}

	topic, err := makeTopic(key)
	if err != nil {
		return err
	}

	index, err := db.indexer.AcquireNext(db.ctx, topic)
	if err != nil {
		return err
	}

	defer db.indexer.Release(topic, index)

	socID, err := client.FeedID(topic, index)
	if err != nil {
		return err
	}

	uploadResp := <-uploadRespC
	if err := uploadResp.err; err != nil {
		return err
	}

	payload := client.PayloadWithTime(uploadResp.ref.Bytes(), time.Unix(0, 0))

	data, sig, err := client.SignSocData(socID, payload, db.privateKey)
	if err != nil {
		return err
	}

	_, err = db.beeCli.UploadSoc(db.ctx, db.owner, socID, data, sig, batchID)
	if err != nil {
		return err
	}

	return nil
}

func (db *bzzdb) Delete(key []byte) error {
	return db.Put(key, nil)
}

func (db *bzzdb) Close() error {
	db.ctxCancel()

	return nil
}

type uploadResp struct {
	ref swarm.Address
	err error
}

func (db *bzzdb) uploadAsync(value []byte) <-chan uploadResp {
	respC := make(chan uploadResp, 1)

	if value == nil {
		respC <- uploadResp{ref: swarm.NewAddress(zeroSocData)}

		return respC
	}

	go func() {
		batchID, err := db.postage.CurrentBatchID(db.ctx)
		if err != nil {
			respC <- uploadResp{err: err}

			return
		}

		resp, err := db.beeCli.UploadBytes(db.ctx, value, batchID)
		if err != nil {
			respC <- uploadResp{err: err}

			return
		}

		respC <- uploadResp{ref: resp.Reference}
	}()

	return respC
}

type downloadFn = func(context.Context, swarm.Address) (io.ReadCloser, error)

//nolint:wrapcheck //relax
func (db *bzzdb) downloadAndRead(downloadFn downloadFn, addr swarm.Address) ([]byte, error) {
	respReader, err := downloadFn(db.ctx, addr)
	if err != nil {
		return nil, err
	}

	respData, err := io.ReadAll(respReader)
	if err != nil {
		return nil, err
	}
	defer respReader.Close()

	return respData, nil
}

//nolint:wrapcheck //relax
func makeTopic(key []byte) (client.Topic, error) {
	data := make([]byte, 0, len(key)+len(keyPrefix))
	data = append(data, keyPrefix...)
	data = append(data, key...)

	return crypto.LegacyKeccak256(data)
}
