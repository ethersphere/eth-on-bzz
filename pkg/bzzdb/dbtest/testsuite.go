// Copyright 2019 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

//nolint:cyclop // relax
package dbtest

import (
	"bytes"
	"testing"

	"github.com/ethersphere/eth-on-bzz/pkg/bzzdb"
)

// TestDatabaseSuite runs a suite of tests against a KeyValueStore database
// implementation.
func TestDatabaseSuite(
	t *testing.T,
	New func() bzzdb.KeyValueStore, //nolint:gocritic // relax
) {
	t.Run("KeyValueOperations", func(t *testing.T) {
		db := New()
		defer db.Close()

		key := []byte("foo")

		if got, err := db.Has(key); err != nil {
			t.Error(err)
		} else if got {
			t.Errorf("wrong value: %t", got)
		}

		value := []byte("hello world")
		if err := db.Put(key, value); err != nil {
			t.Error(err)
		}

		if got, err := db.Has(key); err != nil {
			t.Error(err)
		} else if !got {
			t.Errorf("wrong value: %t", got)
		}

		if got, err := db.Get(key); err != nil {
			t.Error(err)
		} else if !bytes.Equal(got, value) {
			t.Errorf("wrong value: %q", got)
		}

		if err := db.Delete(key); err != nil {
			t.Error(err)
		}

		if got, err := db.Has(key); err != nil {
			t.Error(err)
		} else if got {
			t.Errorf("wrong value: %t", got)
		}
	})
}
