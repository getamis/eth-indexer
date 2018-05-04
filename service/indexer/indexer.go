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

	"bytes"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/getamis/sirius/log"
	"github.com/maichain/eth-indexer/common"
	"github.com/maichain/eth-indexer/model"
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

func (idx *indexer) SyncToTarget(ctx context.Context, targetBlock int64) error {
	childCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Get local state from db
	header, stateBlock, err := idx.getLocalState()
	if err != nil {
		return err
	}

	if targetBlock <= header.Number {
		log.Error("Local block number is ahead of target block", "from", header.Number, "target", targetBlock)
		return errors.New(fmt.Sprintf("targetBlock should be greater than %d", header.Number))
	}

	_, _, err = idx.sync(childCtx, header, &types.Header{Number: big.NewInt(targetBlock)}, stateBlock)
	if err != nil {
		log.Error("Failed to sync from ethereum", "from", header.Number, "target", targetBlock, "err", err)
		return err
	}
	return nil
}

// Listen listens the blocks from given blocks
func (idx *indexer) Listen(ctx context.Context, ch chan *types.Header, fromBlock int64, syncMissingBlocks bool) error {
	childCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var lastSync *model.Header
	var stateBlock *model.StateBlock
	if syncMissingBlocks {
		// Get local state from db
		header, localState, err := idx.getLocalState()
		if err != nil {
			return err
		}
		// Set from block number
		if header.Number < fromBlock-1 {
			header = &model.Header{
				Number: fromBlock - 1,
			}
			localState = &model.StateBlock{
				Number: header.Number - 1,
			}
		}

		// Get latest blocks from ethereum
		latestBlock, err := idx.client.BlockByNumber(childCtx, nil)
		if err != nil {
			log.Error("Failed to get latest header from ethereum", "err", err)
			return err
		}
		stateBlock = localState

		// Sync missing blocks from ethereum
		lastSync, stateBlock, err = idx.sync(childCtx, header, latestBlock.Header(), stateBlock)
		if err != nil {
			log.Error("Failed to sync to latest blocks from ethereum", "from", header.Number, "err", err)
			return err
		}
	}

	// Listen new channel events
	_, err := idx.client.SubscribeNewHead(childCtx, ch)
	if err != nil {
		log.Error("Failed to subscribe event for new header from ethereum", "err", err)
		return err
	}

	for {
		select {
		case head := <-ch:
			log.Trace("Got new header", "number", head.Number, "hash", head.Hash().Hex())
			if lastSync == nil {
				lastSync = &model.Header{
					Number: head.Number.Int64() - 1,
					Hash:   head.ParentHash.Bytes(),
				}
				stateBlock = &model.StateBlock{Number: lastSync.Number}
			}
			lastSync, stateBlock, err = idx.sync(childCtx, lastSync, head, stateBlock)
			if err != nil {
				log.Error("Failed to sync to blocks from ethereum", "from", lastSync.Number, "fromHash", common.BytesToHex(lastSync.Hash), "to", head.Number.Int64(), "fromState", stateBlock.Number, "err", err)
				return err
			}
		case <-childCtx.Done():
			return childCtx.Err()
		}
	}
}

func (idx *indexer) getLocalState() (header *model.Header, stateBlock *model.StateBlock, err error) {
	// Get latest header from db
	header, err = idx.manager.LatestHeader()
	if err != nil {
		if common.NotFoundError(err) {
			log.Info("The header db is empty")
			header = &model.Header{
				Number: -1,
			}
			err = nil
		} else {
			log.Error("Failed to get latest header from db", "err", err)
			return
		}
	}

	// Get latest state block from db
	stateBlock, err = idx.manager.LatestStateBlock()
	if err != nil {
		if common.NotFoundError(err) {
			log.Info("The state db is empty")
			stateBlock = &model.StateBlock{
				Number: 0,
			}
			err = nil
		} else {
			log.Error("Failed to get latest state block from db", "err", err)
			return
		}
	}
	return
}

// sync syncs the blocks and header into database
func (idx *indexer) sync(ctx context.Context, from *model.Header, to *types.Header, stateBlock *model.StateBlock) (*model.Header, *model.StateBlock, error) {
	if to.Number.Int64() <= from.Number {
		log.Debug("Discarding older header", "number", to.Number)
		return from, stateBlock, nil
	}

	// Update existing blocks from ethereum to db
	for i := from.Number + 1; i <= to.Number.Int64(); i++ {
		block, err := idx.client.BlockByNumber(ctx, big.NewInt(i))
		if err != nil {
			log.Error("Failed to get block from ethereum", "number", i, "err", err)
			return from, stateBlock, err
		}

		if !bytes.Equal(block.ParentHash().Bytes(), from.Hash) {
			if err = idx.reorg(ctx, block); err != nil {
				log.Error("Failed to reorg", "number", i, "hash", block.Hash().Hex(), "err", err)
				return from, stateBlock, err
			}
		}

		stateBlock, err = idx.addBlockData(ctx, block, stateBlock)
		if err != nil {
			log.Error("Failed to insert block locally", "number", i, "err", err)
			return from, stateBlock, err
		}
		from = common.Header(block)
	}
	return from, stateBlock, nil
}

func (idx *indexer) addBlockData(ctx context.Context, block *types.Block, fromStateBlock *model.StateBlock) (*model.StateBlock, error) {
	blockNumber := block.Number().Int64()
	logger := log.New("number", blockNumber)
	var receipts []*types.Receipt
	for _, tx := range block.Transactions() {
		logger := logger.New("tx", tx.Hash())
		r, err := idx.client.TransactionReceipt(ctx, tx.Hash())
		if err != nil {
			logger.Error("Failed to get receipt from ethereum", "err", err)
			return fromStateBlock, err
		}
		receipts = append(receipts, r)
	}

	err := idx.manager.InsertBlock(block, receipts)
	if err != nil {
		logger.Error("Failed to insert block", "err", err)
		return fromStateBlock, err
	}
	logger.Trace("Inserted block", "hash", block.Hash().Hex(), "txs", len(block.Transactions()))

	// Get modified accounts
	// Noted: we skip dump block or get modified state error because the state db may not exist
	dump := make(map[string]state.DumpDirtyAccount)
	isGenesis := blockNumber == 0
	if isGenesis {
		d, err := idx.client.DumpBlock(ctx, 0)
		if err != nil {
			log.Warn("Failed to get state from ethereum, ignore it", "number", blockNumber, "err", err)
			return fromStateBlock, nil
		}
		for addr, acc := range d.Accounts {
			dump[addr] = state.DumpDirtyAccount{
				Balance: acc.Balance,
				Nonce:   acc.Nonce,
				Storage: acc.Storage,
			}
		}
	} else {
		// This API is only supported on our customized geth.
		dump, err = idx.client.ModifiedAccountStatesByNumber(ctx, uint64(fromStateBlock.Number), block.Number().Uint64())
		if err != nil {
			log.Warn("Failed to get modified accounts from ethereum, ignore it", "from", fromStateBlock.Number, "to", blockNumber, "err", err)
			return fromStateBlock, nil
		}
	}
	logger.Trace("Start to update state", "hash", block.Hash().Hex(), "accounts", len(dump))

	// Update state db
	err = idx.manager.UpdateState(block, dump)
	if err != nil {
		log.Error("Failed to update state to database", "number", blockNumber, "err", err)
		return fromStateBlock, err
	}
	log.Trace("Inserted state", "number", blockNumber, "hash", block.Hash().Hex(), "accounts", len(dump))
	return &model.StateBlock{
		Number: blockNumber,
	}, nil
}

func (idx *indexer) reorg(ctx context.Context, block *types.Block) error {
	log.Trace("Reorg: tracing starts", "from", block.Number(), "hash", block.Hash().Hex())
	var blocks []*types.Block
	for {
		thisBlock, err := idx.client.BlockByHash(ctx, block.ParentHash())
		if err != nil || thisBlock == nil {
			log.Error("Reorg: failed to get block from ethereum", "hash", block.ParentHash().Hex(), "err", err)
			return err
		}
		block = thisBlock
		blocks = append(blocks, block)

		dbHeader, err := idx.manager.GetHeaderByNumber(block.Number().Int64() - 1)
		if err != nil {
			log.Error("Reorg: failed to get header from local db", "number", block.Number().Int64()-1, "err", err)
			return err
		}

		if bytes.Equal(dbHeader.Hash, block.ParentHash().Bytes()) {
			break
		}
	}
	log.Trace("Reorg: tracing stops", "at", block.Number(), "hash", block.Hash().Hex())
	idx.manager.DeleteDataFromBlock(block.Number().Int64())

	// Get local state from db
	_, stateBlock, err := idx.getLocalState()
	if err != nil {
		return err
	}
	for i := len(blocks) - 1; i >= 0; i-- {
		block = blocks[i]
		stateBlock, err = idx.addBlockData(ctx, block, stateBlock)
		if err != nil {
			log.Error("reorg: failed to insert block data", "number", i, "err", err)
			return err
		}
	}
	log.Trace("Reorg: done", "at", block.Number(), "hash", block.Hash().Hex())
	return nil
}
