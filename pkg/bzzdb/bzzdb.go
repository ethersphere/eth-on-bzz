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

const keyPrefix = "bzzdb-"

var (
	errBzzDBNotFound = errors.New("not found")

	//nolint:gochecknoglobals
	zeroSocData = make([]byte, swarm.HashSize)
)

//nolint:wrapcheck //relax
func New(
	privKey *ecdsa.PrivateKey,
	beeCli client.Client,
) (KeyValueStore, error) {
	owner, err := client.OwnerFromKey(privKey)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &bzzdb{
		privKey:   privKey,
		owner:     owner,
		beeCli:    beeCli,
		postage:   postage.New(beeCli),
		ctx:       ctx,
		ctxCancel: cancel,
	}, nil
}

// bzzdb implements ethereum KeyValueStore interface.
type bzzdb struct {
	privKey *ecdsa.PrivateKey
	beeCli  client.Client
	postage postage.Postage
	owner   common.Address

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

	feedIndexResp, err := db.beeCli.FeedIndexLatest(db.ctx, db.owner, topic)
	if err != nil {
		return nil, err
	}

	if feedIndexResp.Current == 0 && feedIndexResp.Next == 0 {
		return nil, errBzzDBNotFound
	}

	ref, err := client.FeedUpdateReference(db.owner, topic, feedIndexResp.Current)
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

	respData, err = db.downloadAndRead(db.beeCli.Download, swarm.NewAddress(respData))
	if err != nil {
		return nil, err
	}

	return respData, nil
}

//nolint:wrapcheck //relax
func (db *bzzdb) Put(key []byte, value []byte) error {
	batchID, err := db.postage.CurrentBatchID(db.ctx)
	if err != nil {
		return err
	}

	var dataRef []byte
	if value == nil {
		dataRef = zeroSocData
	} else {
		uploadResp, err := db.beeCli.Upload(db.ctx, value, batchID)
		if err != nil {
			return err
		}

		dataRef = uploadResp.Reference.Bytes()
	}

	topic, err := makeTopic(key)
	if err != nil {
		return err
	}

	feedIndexResp, err := db.beeCli.FeedIndexLatest(db.ctx, db.owner, topic)
	if err != nil {
		return err
	}

	socID, err := client.FeedID(topic, feedIndexResp.Next)
	if err != nil {
		return err
	}

	payload := client.PayloadWithTime(dataRef, time.Unix(0, 0))

	data, sig, err := client.SignSocData(socID, payload, db.privKey)
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
	data := []byte(keyPrefix + string(key))

	return crypto.LegacyKeccak256(data)
}
