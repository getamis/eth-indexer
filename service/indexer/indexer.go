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

	return idx.syncTo(childCtx, targetBlock, -1)
}

func (idx *indexer) syncTo(ctx context.Context, targetBlock int64, fromBlock int64) (err error) {
	// Get local state from db
	header, err := idx.getLocalState()
	if err != nil {
		return
	}

	// Set from block number
	if header.Number < fromBlock-1 {
		header = &model.Header{
			Number: fromBlock - 1,
		}
	}

	if targetBlock <= header.Number {
		block, err := idx.client.BlockByNumber(ctx, big.NewInt(targetBlock))
		if err != nil {
			log.Error("Failed to get block from ethereum", "number", targetBlock, "err", err)
			return err
		}

		prevHdr, err := idx.manager.GetHeaderByNumber(targetBlock - 1)
		if err != nil {
			log.Error("Reorg: failed to get header for block", "number", targetBlock-1, "err", err)
			return err
		}

		_, err = idx.reorgMaybe(ctx, block, prevHdr, true)
		if err != nil {
			return err
		}

		return nil
	}
	return idx.sync(ctx, header, targetBlock)
}

func (idx *indexer) Listen(ctx context.Context, ch chan *types.Header, fromBlock int64) error {
	childCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	latestBlock, err := idx.client.BlockByNumber(childCtx, nil)
	if err != nil {
		log.Error("Failed to get latest header from ethereum", "err", err)
		return err
	}
	err = idx.syncTo(childCtx, latestBlock.Number().Int64(), fromBlock)
	if err != nil {
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
			log.Trace("Got new header", "number", head.Number, "hash", head.Hash().Hex())
			err = idx.syncTo(childCtx, head.Number.Int64(), -1)
			if err != nil {
				log.Error("Failed to sync to header from ethereum", "number", head.Number, "err", err)
				return err
			}
		case <-childCtx.Done():
			return childCtx.Err()
		}
	}
}

func (idx *indexer) getLocalState() (header *model.Header, err error) {
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
	return
}

// sync syncs the blocks and header into database
func (idx *indexer) sync(ctx context.Context, from *model.Header, to int64) error {
	var prevTd *big.Int
	// Update existing blocks from ethereum to db
	for i := from.Number + 1; i <= to; i++ {
		block, err := idx.client.BlockByNumber(ctx, big.NewInt(i))
		if err != nil {
			log.Error("Failed to get block from ethereum", "number", i, "err", err)
			return err
		}

		td, err := idx.reorgMaybe(ctx, block, from, false)
		if err != nil {
			return err
		}
		if td != nil {
			prevTd = td
		}
		prevTd, err = idx.addBlockData(ctx, block, from, prevTd)
		if err != nil {
			return err
		}
		from = common.Header(block)
	}
	return nil
}

// insertTd calculates and inserts TD for block.
func (idx *indexer) insertTd(block *types.Block, prevTd *big.Int) (*big.Int, error) {
	blockNumber := block.Number().Int64()
	if prevTd == nil {
		ltd, err := idx.manager.GetTd(block.ParentHash().Bytes())
		if err != nil {
			log.Error("Failed to get TD for block", "number", blockNumber-1, "hash", block.ParentHash().Hex())
			return nil, err
		}
		td, ok := new(big.Int).SetString(ltd.Td, 10)
		if !ok || td.Int64() <= 0 {
			log.Error("Failed to parse TD for block", "number", blockNumber-1, "TD", ltd.Td, "hash", block.ParentHash().Hex())
			return nil, errors.New("failed to parse TD " + ltd.Td)
		}
		prevTd = td
	}

	td := big.NewInt(prevTd.Int64())
	td = td.Add(td, block.Difficulty())
	err := idx.manager.InsertTd(block, td)
	if err != nil && !common.DuplicateError(err) {
		log.Error("Failed to insert td for block", "number", blockNumber, "TD", td, "hash", block.Hash().Hex(), "TD", td)
		return nil, err
	}
	log.Trace("Inserted TD for block", "number", blockNumber, "TD", td, "hash", block.Hash().Hex())
	return td, nil
}

// addBlockData inserts TD, header, transactions, receipts and optionally state for block.
func (idx *indexer) addBlockData(ctx context.Context, block *types.Block, from *model.Header, prevTd *big.Int) (*big.Int, error) {
	td, err := idx.insertTd(block, prevTd)
	if err != nil {
		return nil, err
	}
	err = idx.atomicUpdateBlock(ctx, block)
	if err != nil {
		return nil, err
	}
	return td, nil
}

// reorgMaybe checks targetBlock's parent hash is consistent with local db. if not, reorg and returns the highest TD inserted.
func (idx *indexer) reorgMaybe(ctx context.Context, targetBlock *types.Block, prevHdr *model.Header, oldBlock bool) (*big.Int, error) {
	if bytes.Equal(targetBlock.ParentHash().Bytes(), prevHdr.Hash) {
		return nil, nil
	}

	log.Trace("Reorg: tracing starts", "from", targetBlock.Number(), "hash", targetBlock.Hash().Hex())
	var blocks []*types.Block
	if oldBlock {
		// If we already has data for targetBlock in TD, we need to update it atomically too.
		blocks = append(blocks, targetBlock)
	}
	block := targetBlock
	dbBranchHdr := prevHdr
	for {
		thisBlock, err := idx.client.BlockByHash(ctx, block.ParentHash())
		if err != nil || thisBlock == nil {
			log.Error("Reorg: failed to get block from ethereum", "hash", block.ParentHash().Hex(), "err", err)
			return nil, err
		}
		block = thisBlock
		blocks = append(blocks, block)

		dbHeader, err := idx.manager.GetHeaderByNumber(block.Number().Int64() - 1)
		if err != nil {
			log.Error("Reorg: failed to get header from local db", "number", block.Number().Int64()-1, "err", err)
			return nil, err
		}

		if bytes.Equal(dbHeader.Hash, block.ParentHash().Bytes()) {
			break
		}
		dbBranchHdr = dbHeader
	}
	log.Trace("Reorg: tracing stops", "at", block.Number(), "hash", block.Hash().Hex())
	branchBlock := block

	var prevTd *big.Int
	tds := make(map[int64]*big.Int)
	// Eagerly insert TD for other indexer instances
	for i := len(blocks) - 1; i >= 0; i-- {
		block = blocks[i]
		td, err := idx.insertTd(block, prevTd)
		if err != nil {
			return nil, err
		}
		tds[block.Number().Int64()] = td
		prevTd = td
		log.Trace("Reorg: inserted TD for block", "number", block.Number(), "TD", td)
	}

	// Compare TD at the diversion block 'branchBlock'
	newTd := tds[branchBlock.Number().Int64()]
	ltd, err := idx.manager.GetTd(dbBranchHdr.Hash)
	if err != nil {
		log.Error("Reorg: failed to get TD from DB", "number", dbBranchHdr.Number, "hash", common.BytesToHex(dbBranchHdr.Hash))
		return nil, err
	}
	localTd, ok := new(big.Int).SetString(ltd.Td, 10)
	if !ok {
		log.Error("Reorg: failed to parse TD for block", "number", dbBranchHdr.Number, "TD", ltd.Td, "hash", common.BytesToHex(dbBranchHdr.Hash))
		return nil, errors.New("failed to parse TD " + ltd.Td)
	}
	if localTd.Cmp(newTd) >= 0 {
		return nil, nil
	}

	// Now atomically update the reorg'ed blocks
	log.Trace("Reorg: starting at", "number", branchBlock.Number(), "hash", branchBlock.Hash().Hex())
	err = idx.manager.DeleteStateFromBlock(branchBlock.Number().Int64())
	if err != nil {
		log.Error("Failed to delete state from block", "number", "err", err)
		return nil, err
	}

	for i := len(blocks) - 1; i >= 0; i-- {
		block = blocks[i]
		err = idx.atomicUpdateBlock(ctx, block)
		if err != nil {
			log.Error("Reorg: failed to atomically update block data", "number", i, "err", err)
			return nil, err
		}
		log.Trace("Reorg: atomically updated block", "number", block.Number(), "hash", block.Hash().Hex())
	}
	log.Trace("Reorg: done", "at", block.Number(), "hash", block.Hash().Hex())
	if oldBlock {
		return tds[targetBlock.Number().Int64()], nil
	} else {
		return tds[targetBlock.Number().Int64()-1], nil
	}
}

// atomicUpdateBlock updates the block data (header, transactions, receipts, and optionally state) atomically.
func (idx *indexer) atomicUpdateBlock(ctx context.Context, block *types.Block) error {
	receipts, dump, err := idx.getBlockData(ctx, block)
	if err != nil {
		return err
	}

	err = idx.manager.UpdateBlock(block, receipts, *dump)
	if err != nil {
		log.Error("Failed to update block", "number", block.Number(), "err", err)
		return err
	}
	log.Trace("Updated block", "number", block.Number(), "hash", block.Hash().Hex(), "txs", len(block.Transactions()))
	return nil
}

// getBlockData returns the receipts generated in the given block, and state diff since last block
func (idx *indexer) getBlockData(ctx context.Context, block *types.Block) ([]*types.Receipt, *map[string]state.DumpDirtyAccount, error) {
	blockNumber := block.Number().Int64()
	logger := log.New("number", blockNumber)
	var receipts []*types.Receipt
	for _, tx := range block.Transactions() {
		r, err := idx.client.TransactionReceipt(ctx, tx.Hash())
		if err != nil {
			logger.Error("Failed to get receipt from ethereum", "tx", tx.Hash(), "err", err)
			return nil, nil, err
		}
		receipts = append(receipts, r)
	}

	dump := make(map[string]state.DumpDirtyAccount)
	var err error
	isGenesis := blockNumber == 0
	if isGenesis {
		d, err := idx.client.DumpBlock(ctx, 0)
		if err != nil {
			logger.Error("Failed to get state from ethereum","err", err)
			return nil, nil, err
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
		dump, err = idx.client.ModifiedAccountStatesByNumber(ctx, block.NumberU64()-1, block.Number().Uint64())
		if err != nil {
			logger.Error("Failed to get modified accounts from ethereum", "err", err)
			return nil, nil, err
		}
	}

	return receipts, &dump, nil
}
