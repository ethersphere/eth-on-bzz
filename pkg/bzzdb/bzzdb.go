// Copyright 2023 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bzzdb

func New() KeyValueStore {
	return &bzzdb{}
}

type bzzdb struct{}

func (db *bzzdb) Has(key []byte) (bool, error) {
	v, err := db.Get(key)
	if err != nil {
		return false, err
	}

	return v != nil, nil
}

func (db *bzzdb) Get(key []byte) ([]byte, error) {
	return nil, nil
}

func (db *bzzdb) Put(key []byte, value []byte) error {
	return nil
}

func (db *bzzdb) Delete(key []byte) error {
	return nil
}

func (db *bzzdb) Close() error {
	return nil
}
