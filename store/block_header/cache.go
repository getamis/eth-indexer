// Copyright 2018 The eth-indexer Authors
// This file is part of the eth-indexer library.
//
// The eth-indexer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The eth-indexer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the eth-indexer library. If not, see <http://www.gnu.org/licenses/>.

package block_header

import (
	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/getamis/eth-indexer/common"
	"github.com/getamis/eth-indexer/model"
	"github.com/getamis/sirius/log"
	"github.com/go-sql-driver/mysql"
	lru "github.com/hashicorp/golang-lru"
)

const cacheSize = 128

var duplicateErr = &mysql.MySQLError{
	Number: common.ErrCodeDuplicateKey,
}

// USe global cache because we new a store in each db operation...
var (
	tdCache, _        = lru.NewARC(cacheSize)
	blockHashCache, _ = lru.NewARC(cacheSize)
)

type cacheMiddleware struct {
	Store
}

func newCacheMiddleware(store Store) Store {
	return &cacheMiddleware{
		Store: store,
	}
}

func (t *cacheMiddleware) InsertTd(data *model.TotalDifficulty) (err error) {
	key := common.BytesToHex(data.Hash)
	// If in cache, no need to insert again
	_, ok := tdCache.Get(key)
	if ok {
		return duplicateErr
	}

	defer func() {
		if err == nil || common.DuplicateError(err) {
			tdCache.Add(key, data)
		}
	}()
	return t.Store.InsertTd(data)
}

func (t *cacheMiddleware) Insert(data *model.Header) (err error) {
	// We cannot check cache here, because it may be remove by others
	defer func() {
		if err == nil || common.DuplicateError(err) {
			blockHashCache.Add(common.BytesToHex(data.Hash), data)
		}
	}()
	return t.Store.Insert(data)
}

func (t *cacheMiddleware) FindTd(hash []byte) (result *model.TotalDifficulty, err error) {
	key := common.BytesToHex(hash)
	value, ok := tdCache.Get(key)
	if ok {
		td, ok := value.(*model.TotalDifficulty)
		if ok {
			return td, nil
		}
		log.Warn("Failed to convert value to *model.TotalDifficulty", "value", value, "hash", ethCommon.Bytes2Hex(hash))
		tdCache.Remove(key)
	}

	defer func() {
		if err != nil {
			return
		}
		tdCache.Add(key, result)
	}()
	return t.Store.FindTd(hash)
}

func (t *cacheMiddleware) FindBlockByHash(hash []byte) (result *model.Header, err error) {
	key := common.BytesToHex(hash)
	value, ok := blockHashCache.Get(key)
	if ok {
		header, ok := value.(*model.Header)
		if ok {
			return header, nil
		}
		log.Warn("Failed to convert value to *model.Header", "value", value, "hash", ethCommon.Bytes2Hex(hash))
		blockHashCache.Remove(key)
	}

	defer func() {
		if err != nil {
			return
		}
		blockHashCache.Add(key, result)
	}()
	return t.Store.FindBlockByHash(hash)
}

func (t *cacheMiddleware) FindBlockByNumber(blockNumber int64) (result *model.Header, err error) {
	defer func() {
		if err != nil {
			return
		}
		blockHashCache.Add(common.BytesToHex(result.Hash), result)
	}()
	return t.Store.FindBlockByNumber(blockNumber)
}

func (t *cacheMiddleware) FindLatestBlock() (result *model.Header, err error) {
	defer func() {
		if err != nil {
			return
		}
		blockHashCache.Add(common.BytesToHex(result.Hash), result)
	}()
	return t.Store.FindLatestBlock()
}
