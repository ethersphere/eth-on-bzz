// Copyright 2023 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"crypto/ecdsa"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethersphere/bee/pkg/cac"
	"github.com/ethersphere/bee/pkg/crypto"
	"github.com/ethersphere/bee/pkg/soc"
	"github.com/ethersphere/bee/pkg/swarm"
)

var errSocInvalid = fmt.Errorf("SOC is not valid")

//nolint:wrapcheck //relax
func SignSocData(
	id SocID,
	payload []byte,
	privKey *ecdsa.PrivateKey,
) ([]byte, SocSignature, error) {
	signer := crypto.NewDefaultSigner(privKey)

	ch, err := cac.New(payload)
	if err != nil {
		return nil, nil, err
	}

	sch, err := soc.New(id, ch).Sign(signer)
	if err != nil {
		return nil, nil, err
	}

	if !soc.Valid(sch) {
		return nil, nil, errSocInvalid
	}

	chunkData := sch.Data()
	signatureBytes := chunkData[swarm.HashSize : swarm.HashSize+swarm.SocSignatureSize]
	signature := SocSignature(signatureBytes)

	return ch.Data(), signature, nil
}

func RawDataFromSocResp(resp []byte) []byte {
	start := swarm.SpanSize + swarm.HashSize + swarm.SocSignatureSize

	return resp[start:]
}

//nolint:wrapcheck //relax
func FeedID(topic Topic, index uint64) (SocID, error) {
	idx := make([]byte, 8)
	binary.BigEndian.PutUint64(idx, index)

	fid := make([]byte, 0, 8+len(topic))
	fid = append(fid, topic...)
	fid = append(fid, idx...)

	return crypto.LegacyKeccak256(fid)
}

//nolint:wrapcheck //relax
func FeedUpdateReference(owner common.Address, topic Topic, index uint64) ([]byte, error) {
	feedID, err := FeedID(topic, index)
	if err != nil {
		return nil, err
	}

	ownerBytes := owner.Bytes()

	ref := make([]byte, 0, len(feedID)+len(ownerBytes))
	ref = append(ref, feedID...)
	ref = append(ref, ownerBytes...)

	return crypto.LegacyKeccak256(ref)
}

func PayloadWithTime(payload []byte, t time.Time) []byte {
	res := make([]byte, 8, len(payload)+8)
	binary.BigEndian.PutUint64(res, uint64(t.Unix()))

	res = append(res, payload...)

	return res
}

func PayloadStripTime(payload []byte) []byte {
	return payload[8:]
}

//nolint:wrapcheck //relax
func OwnerFromKey(key *ecdsa.PrivateKey) (common.Address, error) {
	signer := crypto.NewDefaultSigner(key)

	return signer.EthereumAddress()
}
