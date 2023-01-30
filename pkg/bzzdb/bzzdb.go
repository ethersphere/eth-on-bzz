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

	"github.com/ethersphere/bee/pkg/swarm"

	"github.com/ethersphere/eth-on-bzz/pkg/client"
	"github.com/ethersphere/eth-on-bzz/pkg/postage"
)

const (
	deletedSOCData = "deleted"
	keyPrefix      = "bzzdb-"
)

var errBzzDBNotFound = errors.New("not found")

func New(
	privKey *ecdsa.PrivateKey,
	beeCli client.Client,
) KeyValueStore {
	ctx, cancel := context.WithCancel(context.Background())

	return &bzzdb{
		privKey:   privKey,
		beeCli:    beeCli,
		postage:   postage.New(beeCli),
		ctx:       ctx,
		ctxCancel: cancel,
	}
}

// bzzdb implements ethereum KeyValueStore interface.
type bzzdb struct {
	privKey *ecdsa.PrivateKey
	beeCli  client.Client
	postage postage.Postage

	//nolint:containedctx // this ctx is need because methods of KeyValueStore
	// interface do not pass down context. Single context is created in New method
	// and reused for all Bee Client calls.
	ctx       context.Context
	ctxCancel context.CancelFunc
}

func (db *bzzdb) Has(key []byte) (bool, error) {
	if _, err := db.Get(key); err != nil {
		if errors.Is(err, errBzzDBNotFound) {
			return false, nil
		}

		if errors.Is(err, client.ErrNotFound) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

//nolint:wrapcheck //relax
func (db *bzzdb) Get(key []byte) ([]byte, error) {
	owner, err := client.OwnerFromKey(db.privKey)
	if err != nil {
		return nil, err
	}

	swarmKey := makeKey(key)

	feedResp, err := db.beeCli.FeedGet(db.ctx, owner, client.Topic(string(swarmKey)))
	if err != nil {
		return nil, err
	}

	addr := feedResp.Current
	if bytes.Equal(addr, []byte(deletedSOCData)) {
		return nil, errBzzDBNotFound
	}

	body, err := db.beeCli.Download(db.ctx, swarm.NewAddress(addr))
	if err != nil {
		return nil, err
	}

	defer body.Close()

	return io.ReadAll(body)
}

//nolint:wrapcheck //relax
func (db *bzzdb) Put(key []byte, value []byte) error {
	batchID, err := db.postage.CurrentBatchID()
	if err != nil {
		return err
	}

	uploadResp, err := db.beeCli.Upload(db.ctx, value, batchID)
	if err != nil {
		return err
	}

	swarmKey := makeKey(key)
	rawData := uploadResp.Reference.Bytes()

	_, sig, owner, err := client.SignSocData(swarmKey, rawData, db.privKey)
	if err != nil {
		return err
	}

	_, err = db.beeCli.UploadSOC(db.ctx, owner, client.SocID(swarmKey), rawData, sig, batchID)
	if err != nil {
		return err
	}

	return nil
}

//nolint:wrapcheck //relax
func (db *bzzdb) Delete(key []byte) error {
	batchID, err := db.postage.CurrentBatchID()
	if err != nil {
		return err
	}

	swarmKey := makeKey(key)
	rawData := []byte(deletedSOCData)

	_, sig, owner, err := client.SignSocData(swarmKey, rawData, db.privKey)
	if err != nil {
		return err
	}

	_, err = db.beeCli.UploadSOC(db.ctx, owner, client.SocID(swarmKey), rawData, sig, batchID)
	if err != nil {
		return err
	}

	return nil
}

func (db *bzzdb) Close() error {
	db.ctxCancel()

	return nil
}

func makeKey(key []byte) []byte {
	return []byte(keyPrefix + string(key))
}
