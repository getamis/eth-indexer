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

package uncle_header

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
	uncleHashCache, _ = lru.NewARC(cacheSize)
)

type cacheMiddleware struct {
	Store
}

func newCacheMiddleware(store Store) Store {
	return &cacheMiddleware{
		Store: store,
	}
}

func (t *cacheMiddleware) Insert(data *model.UncleHeader) (err error) {
	// We cannot check cache here, because it may be remove by others
	defer func() {
		if err == nil || common.DuplicateError(err) {
			uncleHashCache.Add(common.BytesToHex(data.Hash), data)
		}
	}()
	return t.Store.Insert(data)
}

func (t *cacheMiddleware) FindUncleByHash(hash []byte) (result *model.UncleHeader, err error) {
	key := common.BytesToHex(hash)
	value, ok := uncleHashCache.Get(key)
	if ok {
		header, ok := value.(*model.UncleHeader)
		if ok {
			return header, nil
		}
		log.Warn("Failed to convert value to *model.UncleHeader", "value", value, "hash", ethCommon.Bytes2Hex(hash))
		uncleHashCache.Remove(key)
	}

	defer func() {
		if err != nil {
			return
		}
		uncleHashCache.Add(key, result)
	}()
	return t.Store.FindUncleByHash(hash)
}
