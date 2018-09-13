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

var (
	txCache, _            = lru.NewARC(cacheSize)
	tdCache, _            = lru.NewARC(cacheSize)
	blockCache, _         = lru.NewARC(cacheSize)
	blockReceiptsCache, _ = lru.NewARC(cacheSize)
)

type cacheMiddleware struct {
	EthClient
}

func newCacheMiddleware(client EthClient) EthClient {
	return &cacheMiddleware{
		EthClient: client,
	}
}

func (c *cacheMiddleware) BlockByNumber(ctx context.Context, number *big.Int) (result *types.Block, err error) {
	defer func() {
		if err != nil {
			return
		}
		blockCache.Add(result.Hash().Hex(), result)
	}()
	return c.EthClient.BlockByNumber(ctx, number)
}

func (c *cacheMiddleware) BlockByHash(ctx context.Context, hash common.Hash) (result *types.Block, err error) {
	key := hash.Hex()
	value, ok := blockCache.Get(key)
	if ok {
		block, ok := value.(*types.Block)
		if ok {
			return block, nil
		}
		log.Warn("Failed to convert value to *types.Block", "hash", key)
		blockCache.Remove(key)
	}

	defer func() {
		if err != nil {
			return
		}
		blockCache.Add(key, result)
	}()
	return c.EthClient.BlockByHash(ctx, hash)
}

func (c *cacheMiddleware) TransactionByHash(ctx context.Context, hash common.Hash) (result *types.Transaction, isPending bool, err error) {
	key := hash.Hex()
	value, ok := txCache.Get(key)
	if ok {
		tx, ok := value.(*types.Transaction)
		if ok {
			// Always return pending == false
			return tx, false, nil
		}
		log.Warn("Failed to convert value to *types.Transaction", "hash", key)
		txCache.Remove(key)
	}

	defer func() {
		if err != nil {
			return
		}
		txCache.Add(key, result)
	}()
	return c.EthClient.TransactionByHash(ctx, hash)
}

func (c *cacheMiddleware) GetTotalDifficulty(ctx context.Context, hash common.Hash) (result *big.Int, err error) {
	key := hash.Hex()
	value, ok := tdCache.Get(key)
	if ok {
		td, ok := value.(*big.Int)
		if ok {
			return td, nil
		}
		log.Warn("Failed to convert value to *types.Int", "hash", key)
		tdCache.Remove(key)
	}

	defer func() {
		if err != nil {
			return
		}
		tdCache.Add(key, result)
	}()
	return c.EthClient.GetTotalDifficulty(ctx, hash)
}

func (c *cacheMiddleware) GetBlockReceipts(ctx context.Context, hash common.Hash) (result types.Receipts, err error) {
	key := hash.Hex()
	value, ok := blockReceiptsCache.Get(key)
	if ok {
		receipts, ok := value.(types.Receipts)
		if ok {
			return receipts, nil
		}
		log.Warn("Failed to convert value to types.Receipts", "hash", key)
		blockReceiptsCache.Remove(key)
	}

	defer func() {
		if err != nil {
			return
		}
		blockReceiptsCache.Add(key, result)
	}()
	return c.EthClient.GetBlockReceipts(ctx, hash)
}
