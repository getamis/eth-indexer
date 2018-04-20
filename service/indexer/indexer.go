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
package indexer

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/getamis/sirius/log"
	"github.com/maichain/eth-indexer/common"
	"github.com/maichain/eth-indexer/model"
	"github.com/maichain/eth-indexer/service/pb"
	"github.com/maichain/eth-indexer/store"
)

// New news an indexer service
func New(client EthClient, storeManager store.Manager) *indexer {
	return &indexer{
		client:  client,
		manager: storeManager,
	}
}

type indexer struct {
	client  EthClient
	manager store.Manager
}

func (idx *indexer) Listen(ctx context.Context, ch chan *types.Header) error {
	childCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Get latest header from db
	header, err := idx.manager.LatestHeader()
	if err != nil {
		if common.NotFoundError(err) {
			log.Info("The header db is empty")
			header = &pb.BlockHeader{
				Number: -1,
			}
		} else {
			log.Error("Failed to get latest header from db", "err", err)
			return err
		}
	}

	// Get latest state block from db
	stateBlock, err := idx.manager.LatestStateBlock()
	if err != nil {
		if common.NotFoundError(err) {
			log.Info("The state db is empty")
			stateBlock = &model.StateBlock{
				Number: 0,
			}
		} else {
			log.Error("Failed to get latest state block from db", "err", err)
			return err
		}
	}

	// Get latest blocks from ethereum
	latestBlock, err := idx.client.BlockByNumber(childCtx, nil)
	if err != nil {
		log.Error("Failed to get latest header from ethereum", "err", err)
		return err
	}
	lastBlockHeader := latestBlock.Header()

	// Sync missing blocks from ethereum
	stateBlock, err = idx.sync(childCtx, header.Number, header.Hash, lastBlockHeader.Number.Int64(), stateBlock.Number)
	if err != nil {
		log.Error("Failed to sync to latest blocks from ethereum", "from", header.Number, "fromHash", header.Hash, "err", err)
		return err
	}

	// Listen new channel events
	_, err = idx.client.SubscribeNewHead(childCtx, ch)
	if err != nil {
		log.Error("Failed to subscribe event for new header from ethereum", "err", err)
		return err
	}

	for {
		select {
		case head := <-ch:
			log.Trace("Got new header", "number", head.Number, "hash", common.HashHex(head.Hash()))
			stateBlock, err = idx.sync(childCtx, lastBlockHeader.Number.Int64(), common.HashHex(lastBlockHeader.Hash()), head.Number.Int64(), stateBlock.Number)
			if err != nil {
				log.Error("Failed to sync to blocks from ethereum", "from", lastBlockHeader.Number, "fromHash", lastBlockHeader.Hash(), "to", head.Number.Int64(), "fromState", stateBlock.Number, "err", err)
				return err
			}
			lastBlockHeader = head
		case <-childCtx.Done():
			return childCtx.Err()
		}
	}
}

// sync syncs the blocks and header into database
func (idx *indexer) sync(ctx context.Context, from int64, fromHash string, to int64, fromStateBlock int64) (*model.StateBlock, error) {
	// Update existing blocks from ethereum to db
	for i := from + 1; i <= to; i++ {
		block, err := idx.client.BlockByNumber(ctx, big.NewInt(i))
		if err != nil {
			log.Error("Failed to get block from ethereum", "number", i, "err", err)
			return nil, err
		}

		// TODO: How to handle fork case
		// Check whether fork happens
		// if prevHash != utils.Hex(block.Hash()) {
		//
		// } else {
		// }

		var receipts []*types.Receipt
		for _, tx := range block.Transactions() {
			r, err := idx.client.TransactionReceipt(ctx, tx.Hash())
			if err != nil {
				log.Error("Failed to get receipt from ethereum", "number", i, "tx", tx.Hash(), "err", err)
				return nil, err
			}
			receipts = append(receipts, r)
		}

		err = idx.manager.InsertBlock(block, receipts)
		if err != nil {
			log.Error("Failed to insert block", "number", i, "err", err)
			return nil, err
		}
		log.Trace("Inserted block", "number", i, "hash", common.HashHex(block.Hash()), "txs", len(block.Transactions()))

		// Get modified accounts
		// Noted: we skip dump block or get modified state error because the state db may not exist
		var dump *state.Dump
		isGenesis := i == 0
		if isGenesis {
			dump, err = idx.client.DumpBlock(ctx, 0)
			if err != nil {
				log.Warn("Failed to get state from ethereum, ignore it", "number", i, "err", err)
				continue
			}
		} else {
			// This API is only supportted on our customized geth.
			dump, err = idx.client.ModifiedAccountStatesByNumber(ctx, uint64(fromStateBlock), block.Number().Uint64())
			if err != nil {
				log.Warn("Failed to get modified accounts from ethereum, ignore it", "from", fromStateBlock, "to", i, "err", err)
				continue
			}
		}

		// Update state db
		err = idx.manager.UpdateState(block, dump)
		if err != nil {
			log.Error("Failed to update state to database", "number", i, "err", err)
			return nil, err
		}
		log.Trace("Inserted state", "number", i, "hash", common.HashHex(block.Hash()), "accounts", len(dump.Accounts))

		fromStateBlock = i
	}
	return &model.StateBlock{
		Number: fromStateBlock,
	}, nil
}
