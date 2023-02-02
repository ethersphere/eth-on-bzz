// Copyright 2023 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"crypto/ecdsa"
	"crypto/rand"
	"encoding"
	"encoding/binary"
	"encoding/hex"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethersphere/bee/pkg/cac"
	"github.com/ethersphere/bee/pkg/crypto"
	"github.com/ethersphere/bee/pkg/soc"
	"github.com/ethersphere/bee/pkg/swarm"
)

func RandomAddress() (swarm.Address, error) {
	buf := make([]byte, swarm.HashSize)

	_, err := rand.Read(buf)
	if err != nil {
		return swarm.ZeroAddress, fmt.Errorf("failed to make address: %w", err)
	}

	return swarm.NewAddress(buf), nil
}

var errSocInvalid = fmt.Errorf("SOC is not valid")

type id struct {
	topic []byte
	index int
}

var _ encoding.BinaryMarshaler = (*id)(nil)

func (i *id) MarshalBinary() ([]byte, error) {
	idraw := make([]byte, 0, 8+len(i.topic))
	idraw = append(idraw, i.topic...)

	index := make([]byte, 8)
	binary.BigEndian.PutUint64(index, uint64(i.index))
	idraw = append(idraw, index...)

	return crypto.LegacyKeccak256(idraw)
}

//nolint:wrapcheck //relax
func SignSocData(
	id SocID,
	payload []byte,
	privKey *ecdsa.PrivateKey,
) ([]byte, SocSignature, common.Address, error) {
	signer := crypto.NewDefaultSigner(privKey)

	ch, err := cac.New(payload)
	if err != nil {
		return nil, "", common.Address{}, err
	}

	sch, err := soc.New(id, ch).Sign(signer)
	if err != nil {
		return nil, "", common.Address{}, err
	}

	if !soc.Valid(sch) {
		return nil, "", common.Address{}, errSocInvalid
	}

	chunkData := sch.Data()
	signatureBytes := chunkData[swarm.HashSize : swarm.HashSize+swarm.SocSignatureSize]
	signature := SocSignature(hex.EncodeToString(signatureBytes))

	publicKey, err := signer.PublicKey()
	if err != nil {
		return nil, "", common.Address{}, err
	}

	ownerBytes, err := crypto.NewEthereumAddress(*publicKey)
	if err != nil {
		return nil, "", common.Address{}, err
	}

	owner := common.BytesToAddress(ownerBytes)

	return ch.Data(), signature, owner, nil
}

func RawDataFromSOCResp(resp []byte) []byte {
	start := swarm.SpanSize + swarm.HashSize + swarm.SocSignatureSize

	return resp[start:]
}

//nolint:wrapcheck //relax
func OwnerFromKey(privKey *ecdsa.PrivateKey) (common.Address, error) {
	signer := crypto.NewDefaultSigner(privKey)

	publicKey, err := signer.PublicKey()
	if err != nil {
		return common.Address{}, err
	}

	ownerBytes, err := crypto.NewEthereumAddress(*publicKey)
	if err != nil {
		return common.Address{}, err
	}

	owner := common.BytesToAddress(ownerBytes)

	return owner, nil
}
