// Copyright 2018 AMIS Technologies
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package block_header

import (
	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/getamis/sirius/log"
	"github.com/go-sql-driver/mysql"
	lru "github.com/hashicorp/golang-lru"
	"github.com/getamis/eth-indexer/common"
	"github.com/getamis/eth-indexer/model"
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
