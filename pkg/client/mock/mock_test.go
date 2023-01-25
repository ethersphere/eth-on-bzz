// Copyright 2023 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mock_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/ethersphere/eth-on-bzz/pkg/client"
	"github.com/ethersphere/eth-on-bzz/pkg/client/mock"
)

func Test_Mock_Client(t *testing.T) {
	t.Parallel()

	suite.Run(t, &client.TestSuite{
		Fact: mock.NewClient,
	})
}
