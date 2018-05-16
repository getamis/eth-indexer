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
	"bytes"
	"context"
	"errors"
	"math/big"

	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/getamis/sirius/log"
	"github.com/maichain/eth-indexer/common"
	"github.com/maichain/eth-indexer/model"
	"github.com/maichain/eth-indexer/service"
	"github.com/maichain/eth-indexer/store"
)

var (
	//ErrInconsistentLength returns if the length of ERC20 addresses and block numbers is
	ErrInconsistentLength = errors.New("inconsistent length")
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

// Init ensures all tables for erc20 contracts are created
func (idx *indexer) Init(ctx context.Context, addresses []string, numbers []int) error {
	if len(addresses) != len(numbers) {
		log.Error("Inconsistent array length", "addrs", len(addresses), "numbers", len(numbers))
		return ErrInconsistentLength
	}

	for i, addr := range addresses {
		if !ethCommon.IsHexAddress(addr) {
			return service.ErrInvalidAddress
		}
		address := ethCommon.HexToAddress(addr)

		_, err := idx.manager.FindERC20(address)
		// The ERC20 exists, no need to insert again
		if err == nil {
			continue
		}
		// Other database error, return error
		if !common.NotFoundError(err) {
			return err
		}

		erc20, err := idx.client.GetERC20(ctx, address, int64(numbers[i]))
		if err != nil {
			log.Error("Failed to get ERC20", "addr", addr, "err", err)
			return err
		}

		// Insert ERC20
		err = idx.manager.InsertERC20(erc20)
		if err != nil {
			log.Error("Failed to insert ERC20", "addr", addr, "err", err)
			return err
		}
	}

	return nil
}

// SyncToTarget syncs the blocks fromBlock to targetBlock. In this function, we are NOT checking reorg and inserting TD. We force to INSERT blocks.
func (idx *indexer) SyncToTarget(ctx context.Context, fromBlock, targetBlock int64) error {
	for i := fromBlock; i <= targetBlock; i++ {
		block, err := idx.client.BlockByNumber(ctx, big.NewInt(i))
		if err != nil {
			log.Error("Failed to get block from ethereum", "number", i, "err", err)
			return err
		}
		err = idx.atomicUpdateBlock(ctx, block, idx.manager.ForceInsertBlock)
		if err != nil {
			log.Error("Failed to update block atomically", "number", i, "err", err)
			return err
		}
	}
	return nil
}

func (idx *indexer) Listen(ctx context.Context, ch chan *types.Header) error {
	childCtx, cancel := context.WithCancel(ctx)
	defer cancel()

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
			err = idx.sync(childCtx, -1, head.Number.Int64())
			if err != nil {
				log.Error("Failed to sync to header from ethereum", "number", head.Number, "err", err)
				return err
			}
		case <-childCtx.Done():
			return childCtx.Err()
		}
	}
}

func (idx *indexer) getLocalState() (header *model.Header, currentTd *big.Int, err error) {
	// Get latest header from db
	header, err = idx.manager.LatestHeader()
	if err != nil {
		if common.NotFoundError(err) {
			log.Info("The header db is empty")
			header = &model.Header{
				Number: -1,
				Hash:   ethCommon.Hash{}.Bytes(),
			}
			err = nil
			currentTd = big.NewInt(0)
		} else {
			log.Error("Failed to get latest header from db", "err", err)
			return
		}
	} else {
		ltd, err := idx.manager.GetTd(header.Hash)
		if err != nil {
			log.Error("Failed to get TD from db", "err", err)
			return nil, nil, err
		}
		currentTd, err = common.ParseTd(ltd)
		if err != nil {
			log.Error("Failed to parse TD", "number", ltd.Block, "TD", ltd.Td, "hash", common.BytesToHex(ltd.Hash))
			return nil, nil, err
		}
	}
	return
}

func (idx *indexer) sync(ctx context.Context, from int64, to int64) error {
	// Update existing blocks from ethereum to db
	currHdr, currTd, err := idx.getLocalState()
	if err != nil {
		return err
	}

	// Set from block number
	if currHdr.Number < from-1 {
		currHdr = &model.Header{
			Number: from - 1,
		}
	}

	prevHdr := currHdr
	if to <= currHdr.Number {
		hdr, err := idx.manager.GetHeaderByNumber(to - 1)
		if err != nil {
			log.Error("Failed to get header for block", "number", to-1, "err", err)
			return err
		}
		prevHdr = hdr
	}
	for i := prevHdr.Number + 1; i <= to; i++ {
		block, td, err := idx.addBlockMaybeReorg(ctx, currTd, currHdr, prevHdr, i)
		if err != nil || block == nil {
			return err
		}
		currTd = td
		currHdr = common.Header(block)
		prevHdr = currHdr
	}
	return nil
}

// insertTd calculates and inserts TD for block.
func (idx *indexer) insertTd(block *types.Block, prevTd *big.Int) (*big.Int, error) {
	blockNumber := block.Number().Int64()
	if prevTd == nil {
		ltd, err := idx.manager.GetTd(block.ParentHash().Bytes())
		if err != nil {
			log.Error("Failed to get TD for block", "number", blockNumber-1, "hash", block.ParentHash().Hex(), "err", err)
			return nil, err
		}
		td, err := common.ParseTd(ltd)
		if err != nil {
			log.Error("Failed to parse TD", "number", ltd.Block, "TD", ltd.Td, "hash", common.BytesToHex(ltd.Hash))
			return nil, err
		}
		prevTd = td
	}

	td := big.NewInt(prevTd.Int64())
	td = td.Add(td, block.Difficulty())
	err := idx.manager.InsertTd(block, td)
	if err != nil && !common.DuplicateError(err) {
		log.Error("Failed to insert td for block", "number", blockNumber, "TD", td, "hash", block.Hash().Hex(), "TD", td, "err", err)
		return nil, err
	}
	log.Trace("Inserted TD for block", "number", blockNumber, "TD", td, "hash", block.Hash().Hex())
	return td, nil
}

func (idx *indexer) insertBlocks(ctx context.Context, blocks []*types.Block, tdCache map[int64]*big.Int) (*types.Block, *big.Int, error) {
	var lastTd *big.Int
	var lastInsert *types.Block
	for i := len(blocks) - 1; i >= 0; i-- {
		block := blocks[i]
		if tdCache[block.Number().Int64()] == nil {
			prevTd := tdCache[block.Number().Int64()-1]
			td, err := idx.insertTd(block, prevTd)
			if err != nil {
				return lastInsert, lastTd, err
			}
			lastTd = td
			tdCache[block.Number().Int64()] = td
		}

		err := idx.atomicUpdateBlock(ctx, block, idx.manager.UpdateBlock)
		if err != nil {
			log.Error("Failed to insert block data", "number", block.Number(), "err", err)
			return lastInsert, lastTd, err
		}
		lastInsert = block
	}
	return lastInsert, lastTd, nil
}

// addBlockMaybeReorg checks whether target block's parent hash is consistent with local db.
// if not, reorg and returns the highest TD inserted.
func (idx *indexer) addBlockMaybeReorg(ctx context.Context, currTd *big.Int, currHdr *model.Header, prevHdr *model.Header, target int64) (*types.Block, *big.Int, error) {
	log.Trace("Syncing block", "number", target, "from", prevHdr.Number)
	block, err := idx.client.BlockByNumber(ctx, big.NewInt(target))
	if err != nil {
		log.Error("Failed to get block from ethereum", "number", target, "err", err)
		return nil, currTd, err
	}
	var blocksToInsert []*types.Block
	tdCache := make(map[int64]*big.Int)
	if bytes.Equal(block.ParentHash().Bytes(), prevHdr.Hash) {
		if currHdr.Number >= target {
			return nil, currTd, nil
		}
		blocksToInsert = append(blocksToInsert, block)
		tdCache[currHdr.Number] = currTd
		return idx.insertBlocks(ctx, blocksToInsert, tdCache)
	}

	log.Trace("Reorg tracing: Start", "from", target, "hash", block.Hash().Hex())
	blocks := []*types.Block{block}
	for {
		thisBlock, err := idx.client.BlockByHash(ctx, block.ParentHash())
		if err != nil || thisBlock == nil {
			log.Error("Reorg tracing: Failed to get block from ethereum", "hash", block.ParentHash().Hex(), "err", err)
			return nil, nil, err
		}
		block = thisBlock
		blocks = append(blocks, block)

		dbHeader, err := idx.manager.GetHeaderByNumber(block.Number().Int64() - 1)
		if err != nil {
			log.Error("Reorg tracing: Failed to get header from local db", "number", block.Number().Int64()-1, "err", err)
			return nil, nil, err
		}

		if bytes.Equal(dbHeader.Hash, block.ParentHash().Bytes()) {
			break
		}
	}
	log.Trace("Reorg tracing: Stop", "at", block.Number(), "hash", block.Hash().Hex())
	branchBlock := block

	var prevTd *big.Int
	// Eagerly insert TD for other indexer instances
	for i := len(blocks) - 1; i >= 0; i-- {
		block = blocks[i]
		td, err := idx.insertTd(block, prevTd)
		if err != nil {
			return nil, nil, err
		}
		tdCache[block.Number().Int64()] = td
		prevTd = td
		log.Trace("Reorg tracing: Inserted TD for block", "number", block.Number(), "TD", td)
	}

	// Compare currentTd with the new branch
	newTd := tdCache[target]
	if currTd.Cmp(newTd) >= 0 {
		return nil, currTd, nil
	}
	blocksToInsert = append(blocksToInsert, blocks...)

	// Now atomically update the reorg'ed blocks
	log.Trace("Reorg: Starting at", "number", branchBlock.Number(), "hash", branchBlock.Hash().Hex())
	err = idx.manager.DeleteStateFromBlock(branchBlock.Number().Int64())
	if err != nil {
		log.Error("Reorg: Failed to delete state from block", "number", "err", err)
		return nil, nil, err
	}

	block, _, err = idx.insertBlocks(ctx, blocksToInsert, tdCache)
	if err != nil {
		return block, tdCache[block.Number().Int64()], err
	}
	log.Trace("Reorg: Done", "at", block.Number(), "inserted blocks", len(blocksToInsert), "hash", block.Hash().Hex())
	return block, tdCache[block.Number().Int64()], nil
}

// atomicUpdateBlock updates the block data (header, transactions, receipts, and state) atomically.
func (idx *indexer) atomicUpdateBlock(ctx context.Context, block *types.Block, inserter func(*types.Block, []*types.Receipt, *state.DirtyDump) error) error {
	receipts, dump, err := idx.getBlockData(ctx, block)
	if err != nil {
		return err
	}

	err = inserter(block, receipts, dump)
	if err != nil {
		log.Error("Failed to update block", "number", block.Number(), "err", err)
		return err
	}
	log.Trace("Updated block", "number", block.Number(), "hash", block.Hash().Hex(), "txs", len(block.Transactions()))
	return nil
}

// getBlockData returns the receipts generated in the given block, and state diff since last block
func (idx *indexer) getBlockData(ctx context.Context, block *types.Block) ([]*types.Receipt, *state.DirtyDump, error) {
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

	dump := &state.DirtyDump{}
	var err error
	isGenesis := blockNumber == 0
	if isGenesis {
		d, err := idx.client.DumpBlock(ctx, 0)
		if err != nil {
			logger.Error("Failed to get state from ethereum", "err", err)
			return nil, nil, err
		}
		dump.Root = d.Root
		dump.Accounts = make(map[string]state.DirtyDumpAccount)
		for addr, acc := range d.Accounts {
			dump.Accounts[addr] = state.DirtyDumpAccount{
				Balance: acc.Balance,
				Nonce:   acc.Nonce,
				Storage: acc.Storage,
			}
		}
	} else {
		// This API is only supported on our customized geth.
		dump, err = idx.client.ModifiedAccountStatesByNumber(ctx, block.Number().Uint64())
		if err != nil {
			logger.Error("Failed to get modified accounts from ethereum", "err", err)
			return nil, nil, err
		}
	}

	return receipts, dump, nil
}
