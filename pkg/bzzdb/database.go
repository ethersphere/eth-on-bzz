// Copyright 2023 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bzzdb

import "io"

// KeyValueStore is local interface matching ethereum's ethdb.KeyValueStore.
// Currently this interface only has minimum set of method to cover the most basic
// operations. Over time this interface should expand to fully match
// ethdb.KeyValueStore interface.
type KeyValueStore interface {
	Has(key []byte) (bool, error)
	Get(key []byte) ([]byte, error)
	Put(key []byte, value []byte) error
	Delete(key []byte) error
	io.Closer
}
