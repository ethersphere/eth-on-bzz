// Copyright 2023 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"crypto/rand"
	"fmt"

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
