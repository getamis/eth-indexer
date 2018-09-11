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
	"sync"
	"time"

	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/getamis/eth-indexer/client"
	"github.com/getamis/eth-indexer/common"
	"github.com/getamis/eth-indexer/model"
	"github.com/getamis/eth-indexer/store"
	"github.com/getamis/sirius/log"
)

var (
	//ErrInvalidAddress returns if invalid ERC20 address is detected
	ErrInvalidAddress = errors.New("invalid address")

	retrySubscribeTime = 5 * time.Second
)

// New news an indexer service
func New(clients []client.EthClient, storeManager store.Manager) *indexer {
	return &indexer{
		clients:      clients,
		latestClient: clients[0],
		manager:      storeManager,
	}
}

type indexer struct {
	clients       []client.EthClient
	latestClient  client.EthClient
	manager       store.Manager
	currentHeader *model.Header
	currentTD     *big.Int
}

type Result struct {
	header      *types.Header
	clientIndex int
}

// Init ensures all tables for erc20 contracts are created
func (idx *indexer) SubscribeErc20Tokens(ctx context.Context, addresses []string) error {
	for _, addr := range addresses {
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

		erc20, err := idx.latestClient.GetERC20(ctx, address)
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

	return idx.manager.Init(idx.latestClient)
}

func (idx *indexer) subscribe(ctx context.Context, outChannel chan<- *Result, index int) error {
	client := idx.clients[index]
	ch := make(chan *types.Header)
	childCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	sub, err := client.SubscribeNewHead(childCtx, ch)
	if err != nil {
		log.Warn("Failed to subscribe event for new header", "client", index, "err", err)
		return err
	}

	for {
		select {
		case head := <-ch:
			outChannel <- &Result{
				header:      head,
				clientIndex: index,
			}
		case err := <-sub.Err():
			log.Warn("Receive subscribe error", "client", index, "err", err)
			return err
		case <-childCtx.Done():
			err := childCtx.Err()
			log.Warn("Receive subscribe context error", "client", index, "err", err)
			return err
		}
	}
}

func (idx *indexer) Listen(ctx context.Context, fromBlock int64) error {
	lens := len(idx.clients)
	outChannel := make(chan *Result, 50*lens)
	// Set wait group to cancel
	var wg sync.WaitGroup
	wg.Add(lens)
	defer wg.Wait()

	listenCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// each ethClient will run in different go routines and push messages to channels
	for i := range idx.clients {
		go func(ctx context.Context, ch chan *Result, index int) {
			defer wg.Done()
			for {
				idx.subscribe(ctx, ch, index)
				// Check whether context is done
				if ctx.Err() != nil {
					log.Warn("Listen context error", "client", index, "err", ctx.Err())
					return
				}
				time.Sleep(retrySubscribeTime)
			}
		}(listenCtx, outChannel, i)
	}
	// single thread to process the messages from channel in sequential
	for {
		select {
		case result := <-outChannel:
			header := result.header
			if idx.currentHeader != nil && idx.currentHeader.Number >= header.Number.Int64() {
				log.Trace("Ignore old header", "number", header.Number, "hash", header.Hash().Hex(), "index", result.clientIndex)
				continue
			}

			log.Trace("Got new header", "number", header.Number, "hash", header.Hash().Hex(), "index", result.clientIndex)
			// switch the current ethClient to the source
			idx.latestClient = idx.clients[result.clientIndex]
			err := idx.sync(listenCtx, fromBlock, header)
			if err != nil {
				log.Error("Failed to sync from ethereum", "number", header.Number, "err", err)
				return err
			}
		case <-listenCtx.Done():
			return listenCtx.Err()
		}
	}
}

func (idx *indexer) getLocalState(ctx context.Context, from int64) (*model.Header, *big.Int, error) {
	// Get latest header from db
	header, err := idx.manager.LatestHeader()
	if err != nil {
		if common.NotFoundError(err) {
			log.Warn("The latest header doesn't exist", "from", from)
			return &model.Header{
				Number: from,
			}, big.NewInt(0), nil
		}
		log.Error("Failed to get latest header from db", "err", err)
		return nil, nil, err
	}

	// Get TD
	currentTd, err := idx.getTd(ctx, header.Hash)
	if err != nil {
		log.Error("Failed to get TD", "hash", common.BytesToHex(header.Hash), "err", err)
		return nil, nil, err
	}
	return header, currentTd, nil
}

func (idx *indexer) sync(ctx context.Context, from int64, header *types.Header) error {
	// Update existing blocks and TD from ethereum to db
	var err error
	idx.currentHeader, idx.currentTD, err = idx.getLocalState(ctx, from)
	if err != nil {
		return err
	}

	block, td, err := idx.addBlockMaybeReorg(ctx, header)
	if err != nil {
		return err
	}
	// If the block is inserted, update current td and header
	if block != nil {
		idx.currentHeader = common.Header(block)
		idx.currentTD = td
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
			td, err = idx.latestClient.GetTotalDifficulty(ctx, ethCommon.BytesToHash(hash))
			if err == nil {
				return td, nil
			}
		}

		log.Error("Failed to get TD for block", "hash", ethCommon.Bytes2Hex(hash), "err", err)
		return nil, err
	}
	return common.ParseTd(ltd)
}

func (idx *indexer) insertBlocks(ctx context.Context, blocks []*types.Block, reorgEvent *model.Reorg) (*types.Block, *big.Int, error) {
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
	var events [][]*types.TransferLog
	for i := len(blocks) - 1; i >= 0; i-- {
		block := blocks[i]
		receipt, event, err := idx.getBlockData(ctx, block)
		if err != nil {
			log.Error("Failed to get receipts and state data", "err", err)
			return nil, nil, err
		}
		newBlocks = append(newBlocks, block)
		receipts = append(receipts, receipt)
		events = append(events, event)
	}
	err := idx.manager.UpdateBlocks(ctx, newBlocks, receipts, events, reorgEvent)
	if err != nil {
		log.Error("Failed to update blocks", "err", err)
		return nil, nil, err
	}
	return newBlocks[len(blocks)-1], lastTd, nil
}

// addBlockMaybeReorg checks whether target block's parent hash is consistent with local db.
// if not, reorg and returns the highest TD inserted. Assume target is larger than prevHdr.
func (idx *indexer) addBlockMaybeReorg(ctx context.Context, header *types.Header) (*types.Block, *big.Int, error) {
	logger := log.New("from", idx.currentHeader.Number, "fromHash", common.BytesTo0xHex(idx.currentHeader.Hash), "fromTD", idx.currentTD, "to", header.Number, "toHash", header.Hash().Hex())
	logger.Trace("Syncing block")
	if idx.currentHeader.Number >= header.Number.Int64() {
		logger.Info("Only handle the block with larger block number")
		return nil, nil, nil
	}

	block, err := idx.latestClient.BlockByHash(ctx, header.Hash())
	if err != nil {
		logger.Error("Failed to get block from ethereum", "err", err)
		return nil, nil, err
	}
	// If
	// 1. no block exists in DB
	// 2. on the same chain, we don't need to reorg
	var blocksToInsert []*types.Block
	if idx.currentHeader.Number+1 == header.Number.Int64() && bytes.Equal(block.ParentHash().Bytes(), idx.currentHeader.Hash) {
		blocksToInsert = append(blocksToInsert, block)
		return idx.insertBlocks(ctx, blocksToInsert, nil)
	}

	// Find targetTD to check whether we need to handle it
	targetTD, err := idx.latestClient.GetTotalDifficulty(ctx, header.Hash())
	if err != nil {
		logger.Error("Failed to get target TD", "err", err)
		return nil, nil, err
	}
	logger = logger.New("toTD", targetTD)
	// Compare currentTd with targetTD
	if idx.currentTD.Cmp(targetTD) >= 0 {
		logger.Debug("Ignore this block due to lower TD")
		return nil, nil, nil
	}

	logger.Trace("Reorg tracing: Start")
	reorgEvent := &model.Reorg{
		From:     idx.currentHeader.Number,
		FromHash: idx.currentHeader.Hash,
		To:       idx.currentHeader.Number,
		ToHash:   idx.currentHeader.Hash,
	}
	blocks := []*types.Block{block}
	for {
		// Get old blocks from db only if the number is equal or smaller than the current block number
		if idx.currentHeader.Number == block.Number().Int64()-1 && bytes.Equal(idx.currentHeader.Hash, block.ParentHash().Bytes()) {
			logger.Info("Reorg tracing: Not a reorg event", "number", idx.currentHeader.Number)
			reorgEvent = nil
			break
		} else if idx.currentHeader.Number > block.Number().Int64()-1 {
			dbHeader, err := idx.manager.GetHeaderByNumber(block.Number().Int64() - 1)
			if err != nil {
				if common.NotFoundError(err) {
					reorgEvent = nil
					logger.Warn("Reorg tracing: Clean database", "number", block.Number().Int64()-1)
					break
				}
				logger.Error("Reorg tracing: Failed to get header from local db", "number", block.Number().Int64()-1, "err", err)
				return nil, nil, err
			}
			if bytes.Equal(dbHeader.Hash, block.ParentHash().Bytes()) {
				break
			}
			// Update reorg event
			reorgEvent.From = dbHeader.Number
			reorgEvent.FromHash = dbHeader.Hash
		}

		// Get old blocks from ethereum
		block, err = idx.latestClient.BlockByHash(ctx, block.ParentHash())
		if err != nil || block == nil {
			logger.Error("Reorg tracing: Failed to get block from ethereum", "hash", block.ParentHash().Hex(), "err", err)
			return nil, nil, err
		}
		blocks = append(blocks, block)
	}
	logger.Trace("Reorg tracing: Stop", "at", block.Number(), "hash", block.Hash().Hex())
	branchBlock := block
	blocksToInsert = append(blocksToInsert, blocks...)

	// Now atomically update the reorg'ed blocks
	logger.Trace("Reorg: Starting at", "branch", branchBlock.Number(), "hash", branchBlock.Hash().Hex())
	block, targetTD, err = idx.insertBlocks(ctx, blocksToInsert, reorgEvent)
	if err != nil {
		logger.Error("Reorg: Failed to insert blocks", "err", err)
		return nil, nil, err
	}
	logger.Trace("Reorg: Done", "at", block.Number(), "inserted", len(blocksToInsert), "hash", block.Hash().Hex())
	return block, targetTD, nil
}

// getBlockData returns the receipts generated in the given block, and state diff since last block
func (idx *indexer) getBlockData(ctx context.Context, block *types.Block) ([]*types.Receipt, []*types.TransferLog, error) {
	blockNumber := block.Number().Int64()
	logger := log.New("number", blockNumber, "hash", block.Hash().Hex())

	// Return empty receipts and transfer logs if it's a genesis block
	if blockNumber == 0 {
		return []*types.Receipt{}, []*types.TransferLog{}, nil
	}

	// Get receipts
	receipts, err := idx.latestClient.GetBlockReceipts(ctx, block.Hash())
	if err != nil {
		logger.Error("Failed to get receipts from ethereum", "err", err)
		return nil, nil, err
	}

	// Get Eth transfer events
	events, err := idx.latestClient.GetTransferLogs(ctx, block.Hash())
	if err != nil {
		logger.Error("Failed to get eth transfer events from ethereum", "err", err)
		return nil, nil, err
	}

	return receipts, events, nil
}
