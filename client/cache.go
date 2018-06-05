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

package client

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/getamis/sirius/log"
	lru "github.com/hashicorp/golang-lru"
)

const cacheSize = 128

type cacheMiddleware struct {
	EthClient
	txCache    *lru.ARCCache
	tdCache    *lru.ARCCache
	blockCache *lru.ARCCache
}

func newCacheMiddleware(client EthClient) EthClient {
	txCache, _ := lru.NewARC(cacheSize)
	tdCache, _ := lru.NewARC(cacheSize)
	blockCache, _ := lru.NewARC(cacheSize)

	return &cacheMiddleware{
		EthClient:  client,
		txCache:    txCache,
		tdCache:    tdCache,
		blockCache: blockCache,
	}
}

func (c *cacheMiddleware) BlockByNumber(ctx context.Context, number *big.Int) (result *types.Block, err error) {
	defer func() {
		if err != nil {
			return
		}
		c.blockCache.Add(result.Hash().Hex(), result)
	}()
	return c.EthClient.BlockByNumber(ctx, number)
}

func (c *cacheMiddleware) BlockByHash(ctx context.Context, hash common.Hash) (result *types.Block, err error) {
	key := hash.Hex()
	value, ok := c.blockCache.Get(key)
	if ok {
		block, ok := value.(*types.Block)
		if ok {
			return block, nil
		}
		log.Warn("Failed to convert value to *types.Block", "hash", key)
		c.blockCache.Remove(key)
	}

	defer func() {
		if err != nil {
			return
		}
		c.blockCache.Add(key, result)
	}()
	return c.EthClient.BlockByHash(ctx, hash)
}

func (c *cacheMiddleware) TransactionByHash(ctx context.Context, hash common.Hash) (result *types.Transaction, isPending bool, err error) {
	key := hash.Hex()
	value, ok := c.txCache.Get(key)
	if ok {
		tx, ok := value.(*types.Transaction)
		if ok {
			// Always return pending == false
			return tx, false, nil
		}
		log.Warn("Failed to convert value to *types.Transaction", "hash", key)
		c.txCache.Remove(key)
	}

	defer func() {
		if err != nil {
			return
		}
		c.txCache.Add(key, result)
	}()
	return c.EthClient.TransactionByHash(ctx, hash)
}

func (c *cacheMiddleware) GetTotalDifficulty(ctx context.Context, hash common.Hash) (result *big.Int, err error) {
	key := hash.Hex()
	value, ok := c.tdCache.Get(key)
	if ok {
		td, ok := value.(*big.Int)
		if ok {
			return td, nil
		}
		log.Warn("Failed to convert value to *types.Int", "hash", key)
		c.tdCache.Remove(key)
	}

	defer func() {
		if err != nil {
			return
		}
		c.tdCache.Add(key, result)
	}()
	return c.EthClient.GetTotalDifficulty(ctx, hash)
}
