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
	"strconv"

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

type ethResult struct {
	head        *types.Header
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

func (idx *indexer) Listen(ctx context.Context, ch chan *types.Header, fromBlock int64) error {
	childCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// outChannel collects heads from ethClients
	outChannel := make(chan ethResult)
	// errChannel receives the errors from ethClients and decides whether to restart the indexer by thowing out the error
	errChannel := make(chan error)
	// cancelChannel simply terminates the indexer when receiving message
	cancelChannel := make(chan error)

	// each ethClient will run in different go routines and push messages to channels
	for i, c := range idx.clients {
		go func(index int, client client.EthClient) {
			sub, err := client.SubscribeNewHead(childCtx, ch)

			if err != nil {
				log.Warn("Failed to subscribe event for new header from EthClient ["+strconv.Itoa(index)+"].", "warn", err)
				errChannel <- err
				if sub != nil {
					sub.Unsubscribe()
				}
				return
			}

			for {
				select {
				case head := <-ch:
					result := ethResult{
						head:        head,
						clientIndex: index,
					}
					outChannel <- result
				case <-childCtx.Done():
					cancelChannel <- childCtx.Err()
					return
				case err := <-sub.Err():
					// Err returns the subscription error channel. The error channel receives
					// a value if there is an issue with the subscription (e.g. the network connection
					// delivering the events has been closed). Only one value will ever be sent.
					// The error channel is closed by Unsubscribe.
					log.Warn("Failed to subscribe for new header from EthClient ["+strconv.Itoa(index)+"].", "warn", err)
					errChannel <- err
					sub.Unsubscribe()
				}
			}
		}(i, c)
	}
	// failureCount records the total errors we have from subscribing ethClients
	// strong assumption: everyClient will return sub.Err() once
	failureCount := 0
	// single thread to process the messages from channel in sequential
	for {
		select {
		case result := <-outChannel:
			if fromBlock > result.head.Number.Int64() {
				log.Trace("Ignore old header", "number", result.head.Number, "hash", result.head.Hash().Hex())
				continue
			}
			log.Trace("Got new header", "number", result.head.Number, "hash", result.head.Hash().Hex(), "index", result.clientIndex)
			// switch the current ethClient to the source
			idx.latestClient = idx.clients[result.clientIndex]
			err := idx.sync(childCtx, fromBlock, result.head.Number.Int64())
			if err != nil {
				log.Error("Failed to sync from ethereum", "number", result.head.Number, "err", err)
				return err
			}
		case err := <-errChannel:
			failureCount++
			if failureCount >= len(idx.clients) {
				log.Error("Failed to subscribe event from all the "+strconv.Itoa(len(idx.clients))+" EthClient(s) we have.", "err", err)
				return err
			}
		case canceled := <-cancelChannel:
			return canceled
		}
	}
	return nil
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
		block, err := idx.latestClient.BlockByNumber(ctx, big.NewInt(from-1))
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
func (idx *indexer) addBlockMaybeReorg(ctx context.Context, target int64) (*types.Block, *big.Int, error) {
	logger := log.New("from", idx.currentHeader.Number, "fromTD", idx.currentTD, "to", target)
	logger.Trace("Syncing block")
	block, err := idx.latestClient.BlockByNumber(ctx, big.NewInt(target))
	if err != nil {
		logger.Error("Failed to get block from ethereum", "err", err)
		return nil, nil, err
	}

	// Check total difficulty to see if the block with the same height has been different
	// e.g, given the previous latestClient geth1, and current latestClient geth2:
	//      with the same block number 1001 are actually different blocks for geth1 and geth2 ( which means fork has happened,)
	//      we suppose to have different total difficulties and need to deal with blocks data reorg
	_, err = idx.getTd(ctx, block.Hash().Bytes())
	if err == nil {
		logger.Info("Block is already in our indexer database", "number", block.Number().Int64())
		return nil, nil, nil
	}

	// If on the same chain, we don't need to reorg
	var blocksToInsert []*types.Block
	if target == 0 || bytes.Equal(block.ParentHash().Bytes(), idx.currentHeader.Hash) {
		blocksToInsert = append(blocksToInsert, block)
		return idx.insertBlocks(ctx, blocksToInsert, nil)
	}

	logger.Trace("Reorg tracing: Start")
	reorgEvent := &model.Reorg{
		From:     idx.currentHeader.Number,
		FromHash: idx.currentHeader.Hash,
		To:       idx.currentHeader.Number,
		ToHash:   idx.currentHeader.Hash,
	}
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
				// Update reorg event
				reorgEvent.From = dbHeader.Number
				reorgEvent.FromHash = dbHeader.Hash
			} else if !common.NotFoundError(err) {
				logger.Error("Reorg tracing: Failed to get header from local db", "number", block.Number().Int64()-1, "err", err)
				return nil, nil, err
			}
			// Ignore not found error
		}

		// Get old blocks from ethereum
		block, err = idx.latestClient.BlockByHash(ctx, block.ParentHash())
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
	if idx.currentTD.Cmp(targetTD) > 0 {
		logger.Debug("Ignore this block due to lower TD", "targetTD", targetTD)
		return nil, nil, nil
	}
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
