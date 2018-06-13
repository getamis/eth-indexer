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

package indexer

import (
	"bytes"
	"context"
	"errors"
	"math/big"

	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/getamis/eth-indexer/client"
	"github.com/getamis/eth-indexer/common"
	"github.com/getamis/eth-indexer/model"
	"github.com/getamis/eth-indexer/store"
	"github.com/getamis/sirius/log"
)

var (
	//ErrInconsistentLength returns if the length of ERC20 addresses and block numbers are not eqaul
	ErrInconsistentLength = errors.New("inconsistent length")
	//ErrInvalidAddress returns if invalid ERC20 address is detected
	ErrInvalidAddress = errors.New("invalid address")
)

// New news an indexer service
func New(client client.EthClient, storeManager store.Manager) *indexer {
	return &indexer{
		client:  client,
		manager: storeManager,
	}
}

type indexer struct {
	client        client.EthClient
	manager       store.Manager
	currentHeader *model.Header
	currentTD     *big.Int
}

// Init ensures all tables for erc20 contracts are created
func (idx *indexer) Init(ctx context.Context, addresses []string, numbers []int) error {
	if len(addresses) != len(numbers) {
		log.Error("Inconsistent array length", "addrs", len(addresses), "numbers", len(numbers))
		return ErrInconsistentLength
	}

	for i, addr := range addresses {
		if !ethCommon.IsHexAddress(addr) {
			return ErrInvalidAddress
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

	return idx.manager.Init()
}

// SyncToTarget syncs the blocks fromBlock to targetBlock. In this function, we are NOT checking reorg and inserting TD. We force to INSERT blocks.
func (idx *indexer) SyncToTarget(ctx context.Context, fromBlock, targetBlock int64) error {
	for i := fromBlock; i <= targetBlock; i++ {
		block, err := idx.client.BlockByNumber(ctx, big.NewInt(i))
		if err != nil {
			log.Error("Failed to get block from ethereum", "number", i, "err", err)
			return err
		}
		_, _, err = idx.insertBlocks(ctx, []*types.Block{block}, store.ModeForceSync)
		if err != nil {
			log.Error("Failed to update block atomically", "number", i, "err", err)
			return err
		}
	}
	return nil
}

func (idx *indexer) Listen(ctx context.Context, ch chan *types.Header, fromBlock int64) error {
	childCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Listen new channel events
	sub, err := idx.client.SubscribeNewHead(childCtx, ch)
	if err != nil {
		log.Error("Failed to subscribe event for new header from ethereum", "err", err)
		return err
	}

	for {
		select {
		case head := <-ch:
			log.Trace("Got new header", "number", head.Number, "hash", head.Hash().Hex())
			err = idx.sync(childCtx, fromBlock, head.Number.Int64())
			if err != nil {
				log.Error("Failed to sync to header from ethereum", "number", head.Number, "err", err)
				return err
			}
		case <-childCtx.Done():
			return childCtx.Err()
		case err := <-sub.Err():
			log.Error("Failed to subscribe new chain head", "err", err)
			return err
		}
	}
}

func (idx *indexer) getLocalState(ctx context.Context, from int64) (header *model.Header, currentTd *big.Int, err error) {
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
	}

	// Ensure the from block is lager than current block
	if from-1 > header.Number {
		block, err := idx.client.BlockByNumber(ctx, big.NewInt(from-1))
		if err != nil {
			log.Error("Failed to get block", "number", from, "err", err)
			return nil, nil, err
		}
		header = common.Header(block)
	}

	if header.Number >= 0 {
		currentTd, err = idx.getTd(ctx, header.Hash)
		if err != nil {
			log.Error("Failed to get TD", "hash", common.BytesToHex(header.Hash), "err", err)
			return nil, nil, err
		}
	}
	return
}

func (idx *indexer) sync(ctx context.Context, from int64, to int64) error {
	// Update existing blocks from ethereum to db
	var err error
	idx.currentHeader, idx.currentTD, err = idx.getLocalState(ctx, from)
	if err != nil {
		return err
	}

	// Ensure the from block is lager than current block
	from = idx.currentHeader.Number + 1

	if from > to {
		// Only check `to` block
		from = to
	}

	for i := from; i <= to; i++ {
		block, td, err := idx.addBlockMaybeReorg(ctx, i)
		if err != nil {
			return err
		}
		// If a block is inserted, update current td and header
		if block != nil {
			idx.currentHeader = common.Header(block)
			idx.currentTD = td
		}
	}
	return nil
}

// insertTd calculates and inserts TD for block.
func (idx *indexer) insertTd(ctx context.Context, block *types.Block) (*big.Int, error) {
	blockNumber := block.Number().Int64()

	// Check whether it's a genesis block
	var prevTd *big.Int
	var err error
	if blockNumber == 0 {
		prevTd = ethCommon.Big0
	} else {
		parentHash := block.ParentHash()
		prevTd, err = idx.getTd(ctx, parentHash.Bytes())
		if err != nil {
			log.Error("Failed to get TD", "number", blockNumber-1, "hash", parentHash.Hex(), "err", err)
			return nil, err
		}
	}

	td := new(big.Int).Add(prevTd, block.Difficulty())
	err = idx.manager.InsertTd(block, td)
	if err != nil && !common.DuplicateError(err) {
		log.Error("Failed to insert td for block", "number", blockNumber, "TD", td, "hash", block.Hash().Hex(), "TD", td, "err", err)
		return nil, err
	}
	log.Trace("Inserted TD for block", "number", blockNumber, "TD", td, "hash", block.Hash().Hex())
	return td, nil
}

// getTd gets td from db, and try to get from ethereum if db not found.
func (idx *indexer) getTd(ctx context.Context, hash []byte) (td *big.Int, err error) {
	ltd, err := idx.manager.GetTd(hash)

	if err != nil {
		// If not found, try to get it from ethereum
		if common.NotFoundError(err) {
			log.Warn("Failed to get TD from db, try to get it from ethereum", "hash", ethCommon.Bytes2Hex(hash), "err", err)
			td, err = idx.client.GetTotalDifficulty(ctx, ethCommon.BytesToHash(hash))
			if err == nil {
				return td, nil
			}
		}

		log.Error("Failed to get TD for block", "hash", ethCommon.Bytes2Hex(hash), "err", err)
		return nil, err
	}
	return common.ParseTd(ltd)
}

func (idx *indexer) insertBlocks(ctx context.Context, blocks []*types.Block, mode store.UpdateMode) (*types.Block, *big.Int, error) {
	var lastTd *big.Int
	// Insert td
	for i := len(blocks) - 1; i >= 0; i-- {
		td, err := idx.insertTd(ctx, blocks[i])
		if err != nil {
			return nil, nil, err
		}
		lastTd = td
	}

	// Update blocks
	var newBlocks []*types.Block
	var receipts [][]*types.Receipt
	var dumps []*state.DirtyDump
	var events [][]*types.TransferLog
	for i := len(blocks) - 1; i >= 0; i-- {
		block := blocks[i]
		receipt, dump, event, err := idx.getBlockData(ctx, block)
		if err != nil {
			log.Error("Failed to get receipts and state data", "err", err)
			return nil, nil, err
		}
		newBlocks = append(newBlocks, block)
		receipts = append(receipts, receipt)
		dumps = append(dumps, dump)
		events = append(events, event)
	}
	err := idx.manager.UpdateBlocks(newBlocks, receipts, dumps, events, mode)
	if err != nil {
		log.Error("Failed to update blocks", "err", err)
		return nil, nil, err
	}
	return newBlocks[len(blocks)-1], lastTd, nil
}

// addBlockMaybeReorg checks whether target block's parent hash is consistent with local db.
// if not, reorg and returns the highest TD inserted. Assume target is larger than prevHdr.
func (idx *indexer) addBlockMaybeReorg(ctx context.Context, target int64) (*types.Block, *big.Int, error) {
	logger := log.New("from", idx.currentHeader.Number, "fromTD", idx.currentTD, "to", target)
	logger.Trace("Syncing block")
	block, err := idx.client.BlockByNumber(ctx, big.NewInt(target))
	if err != nil {
		logger.Error("Failed to get block from ethereum", "err", err)
		return nil, nil, err
	}

	// If in the same chain, we don't need to check reorg
	var blocksToInsert []*types.Block
	if target == 0 || bytes.Equal(block.ParentHash().Bytes(), idx.currentHeader.Hash) {
		blocksToInsert = append(blocksToInsert, block)
		return idx.insertBlocks(ctx, blocksToInsert, store.ModeSync)
	}

	logger.Trace("Reorg tracing: Start")
	targetTD := block.Difficulty()
	blocks := []*types.Block{block}
	for {
		// Get old blocks from db only if the number is not equal to current block number
		if idx.currentHeader.Number != block.Number().Int64()-1 {
			dbHeader, err := idx.manager.GetHeaderByNumber(block.Number().Int64() - 1)
			if err == nil {
				if bytes.Equal(dbHeader.Hash, block.ParentHash().Bytes()) {
					break
				}
			} else if !common.NotFoundError(err) {
				logger.Error("Reorg tracing: Failed to get header from local db", "number", block.Number().Int64()-1, "err", err)
				return nil, nil, err
			}
			// Ignore not found error
		}

		// Get old blocks from ethereum
		block, err = idx.client.BlockByHash(ctx, block.ParentHash())
		if err != nil || block == nil {
			logger.Error("Reorg tracing: Failed to get block from ethereum", "hash", block.ParentHash().Hex(), "err", err)
			return nil, nil, err
		}
		blocks = append(blocks, block)
		targetTD.Add(targetTD, block.Difficulty())
	}
	logger.Trace("Reorg tracing: Stop", "at", block.Number(), "hash", block.Hash().Hex())
	branchBlock := block

	// Calculate target TD
	rootTD, err := idx.getTd(ctx, branchBlock.ParentHash().Bytes())
	if err != nil {
		logger.Error("Reorg tracing: Failed to get TD", "hash", branchBlock.ParentHash().Hex(), "err", err)
		return nil, nil, err
	}
	targetTD.Add(targetTD, rootTD)

	// Compare currentTd with the new branch
	if idx.currentTD.Cmp(targetTD) >= 0 {
		logger.Debug("Ignore this block due to lower TD", "targetTD", targetTD)
		return nil, nil, nil
	}
	blocksToInsert = append(blocksToInsert, blocks...)

	// Now atomically update the reorg'ed blocks
	logger.Trace("Reorg: Starting at", "branch", branchBlock.Number(), "hash", branchBlock.Hash().Hex())
	block, targetTD, err = idx.insertBlocks(ctx, blocksToInsert, store.ModeReOrg)
	if err != nil {
		logger.Error("Reorg: Failed to insert blocks", "err", err)
		return nil, nil, err
	}
	logger.Trace("Reorg: Done", "at", block.Number(), "inserted", len(blocksToInsert), "hash", block.Hash().Hex())
	return block, targetTD, nil
}

// getBlockData returns the receipts generated in the given block, and state diff since last block
func (idx *indexer) getBlockData(ctx context.Context, block *types.Block) ([]*types.Receipt, *state.DirtyDump, []*types.TransferLog, error) {
	blockNumber := block.Number().Int64()
	logger := log.New("number", blockNumber)

	// Get receipts
	receipts, err := idx.client.TransactionReceipts(ctx, block.Transactions())
	if err != nil {
		logger.Error("Failed to get receipts from ethereum", "err", err)
		return nil, nil, nil, err
	}

	// Get state dump
	dump := &state.DirtyDump{}
	isGenesis := blockNumber == 0
	if isGenesis {
		d, err := idx.client.DumpBlock(ctx, 0)
		if err != nil {
			logger.Error("Failed to get state from ethereum", "err", err)
			return nil, nil, nil, err
		}
		dump.Root = d.Root
		dump.Accounts = make(map[string]state.DirtyDumpAccount)
		for addr, acc := range d.Accounts {
			dump.Accounts[addr] = state.DirtyDumpAccount{
				Balance: &acc.Balance,
				Storage: acc.Storage,
			}
		}
	} else {
		// This API is only supported on our customized geth.
		dump, err = idx.client.ModifiedAccountStatesByNumber(ctx, block.Number().Uint64())
		if err != nil {
			logger.Error("Failed to get modified accounts from ethereum", "err", err)
			return nil, nil, nil, err
		}
	}

	// Get Eth transfer events
	events, err := idx.client.GetTransferLogs(ctx, block.Hash())
	if err != nil {
		logger.Error("Failed to get eth transfer events from ethereum", "err", err)
		return nil, nil, nil, err
	}

	return receipts, dump, events, nil
}
