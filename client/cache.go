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
	blockCache *lru.ARCCache
}

func newCacheMiddleware(client EthClient) EthClient {
	txCache, _ := lru.NewARC(cacheSize)
	blockCache, _ := lru.NewARC(cacheSize)

	return &cacheMiddleware{
		EthClient:  client,
		txCache:    txCache,
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
