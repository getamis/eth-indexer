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

package store

import (
	"bytes"
	"context"
	"errors"
	"math/big"

	gethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/getamis/eth-indexer/client"
	"github.com/getamis/eth-indexer/common"
	"github.com/getamis/eth-indexer/model"
	"github.com/getamis/eth-indexer/store/account"
	"github.com/getamis/eth-indexer/store/block_header"
	"github.com/getamis/eth-indexer/store/reorg"
	. "github.com/getamis/eth-indexer/store/sqldb"
	"github.com/getamis/eth-indexer/store/subscription"
	"github.com/getamis/eth-indexer/store/transaction"
	"github.com/getamis/eth-indexer/store/transaction_receipt"
	"github.com/getamis/sirius/log"
	"github.com/jmoiron/sqlx"
)

//go:generate mockery -name Manager

// Manager is a wrapper interface to insert block, receipt and states quickly
type Manager interface {
	// Init the store manager to load the erc20 list
	Init(ctx context.Context) error
	// FindERC20 finds the erc20 code
	FindERC20(ctx context.Context, address gethCommon.Address) (*model.ERC20, error)
	// InsertERC20 inserts the erc20 code
	InsertERC20(ctx context.Context, code *model.ERC20) error
	// InsertTd writes the total difficulty for a block
	InsertTd(ctx context.Context, data *model.TotalDifficulty) error
	// FindLatestBlock returns a latest header from db
	FindLatestBlock(ctx context.Context) (*model.Header, error)
	// FindBlockByNumber returns the header of the given block number
	FindBlockByNumber(ctx context.Context, number int64) (*model.Header, error)
	// FindTd returns the TD of the given block hash
	FindTd(ctx context.Context, hash []byte) (*model.TotalDifficulty, error)
	// InsertBlocks updates all block data
	InsertBlocks(ctx context.Context, balancer client.Balancer, blocks []*types.Block, receipts [][]*types.Receipt, events [][]*types.TransferLog) error
	// ReorgBlocks inserts reorg event and deletes the forked block data.
	// To improve the deletion performance, we delete block data by `deleteChunk` size.
	// The chunk solution is referenced by http://mysql.rjweb.org/doc.php/deletebig.
	ReorgBlocks(ctx context.Context, reorgEvent *model.Reorg) error
}

type headerStore = block_header.Store
type accountStore = account.Store

var (
	// ErrModifiedData returns if the data is modified by others. Update your local states if we received the error.
	ErrModifiedData = errors.New("modified data")
	// deleteBlocksChunk defines the block size we delete at once
	deleteBlocksChunk = int64(20)
)

type manager struct {
	// Stores
	headerStore
	accountStore

	db          *sqlx.DB
	chainConfig *params.ChainConfig
	tokenList   map[gethCommon.Address]*model.ERC20
}

// NewManager news a store manager to insert block, receipts and states.
func NewManager(db *sqlx.DB, chainConfig *params.ChainConfig) Manager {
	return &manager{
		db:           db,
		headerStore:  block_header.NewWithDB(db, block_header.Cache()),
		accountStore: account.NewWithDB(db),
		chainConfig:  chainConfig,
	}
}

func (m *manager) Init(ctx context.Context) error {
	list, err := account.NewWithDB(m.db).ListOldERC20(ctx)
	if err != nil && !common.NotFoundError(err) {
		return err
	}
	tokenList := make(map[gethCommon.Address]*model.ERC20, len(list))
	tokenList[model.ETHAddress] = &model.ERC20{
		Address:     model.ETHBytes,
		BlockNumber: 0,
	}
	for _, e := range list {
		tokenList[gethCommon.BytesToAddress(e.Address)] = e
	}
	m.tokenList = tokenList
	return nil
}

func (m *manager) InsertBlocks(ctx context.Context, balancer client.Balancer, blocks []*types.Block, receipts [][]*types.Receipt, events [][]*types.TransferLog) (err error) {
	blockSize := len(blocks)
	if blockSize != len(receipts) || blockSize != len(events) {
		log.Error("Inconsistent states", "blocks", blockSize, "receipts", len(receipts))
		return common.ErrInconsistentStates
	}

	from := int64(blocks[0].NumberU64())
	to := int64(blocks[blockSize-1].NumberU64())
	if (to - from + 1) != int64(blockSize) {
		log.Error("Inconsistent size and range", "size", blockSize, "range", to-from+1)
		return common.ErrInconsistentStates
	}

	dbTx, err := m.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			dbTx.Rollback()
			return
		}
		err = dbTx.Commit()
	}()

	// Start to insert blocks and states
	for i := 0; i < blockSize; i++ {
		err = m.insertBlock(ctx, dbTx, balancer, blocks[i], receipts[i], events[i])
		if err != nil {
			return err
		}
	}

	headerStore := block_header.NewWithDB(dbTx, block_header.Cache())
	// Ensure the parent block of block 0 exists
	header, err := headerStore.FindBlockByNumber(ctx, from-1)
	if err == nil {
		if !bytes.Equal(header.Hash, blocks[0].ParentHash().Bytes()) {
			log.Warn("Inconsistent parent header", "db", common.BytesTo0xHex(header.Hash), "expected", blocks[0].ParentHash().Hex())
			return ErrModifiedData
		}
		return nil
	}
	// If not found, check if it is the first insertion.
	if common.NotFoundError(err) {
		blockCount, err := headerStore.CountBlocks(ctx)
		if err != nil {
			return err
		}
		if blockCount == uint64(blockSize) {
			return nil
		}
		log.Warn("Parent header is not found")
		return ErrModifiedData
	}

	return err
}

func (m *manager) ReorgBlocks(ctx context.Context, reorgEvent *model.Reorg) (err error) {
	if reorgEvent == nil {
		return nil
	}

	dbTx, err := m.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			dbTx.Rollback()
			return
		}
		err = dbTx.Commit()
	}()

	reorgStore := reorg.NewWithDB(dbTx)
	err = reorgStore.Insert(ctx, reorgEvent)
	if err != nil {
		return err
	}

	// Delete blocks from latest to oldest blocks
	end := reorgEvent.To
	for begin := reorgEvent.To - deleteBlocksChunk + 1; end >= reorgEvent.From; begin -= deleteBlocksChunk {
		if begin < reorgEvent.From {
			begin = reorgEvent.From
		}
		err = m.delete(ctx, dbTx, begin, end)
		if err != nil {
			return err
		}
		end = begin - 1
	}

	return nil
}

// insertBlock inserts block, and accounts inside a DB transaction
func (m *manager) insertBlock(ctx context.Context, dbTx DbOrTx, balancer client.Balancer, block *types.Block, receipts []*types.Receipt, ethEvents []*types.TransferLog) (err error) {
	headerStore := block_header.NewWithDB(dbTx, block_header.Cache())
	txStore := transaction.NewWithDB(dbTx)
	receiptStore := transaction_receipt.NewWithDB(dbTx)
	accountStore := account.NewWithDB(dbTx)
	subsStore := subscription.NewWithDB(dbTx)
	blockNumber := block.Number().Int64()
	txPrices := make(map[gethCommon.Hash]*big.Int)
	totalTxsFee := new(big.Int)

	// Insert txs
	var txs []*model.Transaction
	for _, t := range block.Transactions() {
		tx, err := common.Transaction(m.chainConfig, block, t)
		if err != nil {
			return err
		}
		txs = append(txs, tx)
		err = txStore.Insert(ctx, tx)
		if err != nil {
			return err
		}
		txPrices[t.Hash()] = t.GasPrice()
	}

	var events []*model.Transfer
	// Add eth transfer events
	for _, e := range ethEvents {
		events = append(events, common.EthTransferEvent(block, e))
	}

	// Insert tx receipts
	for _, receipt := range receipts {
		r, err := common.Receipt(block, receipt)
		if err != nil {
			return err
		}

		err = receiptStore.Insert(ctx, r)
		if err != nil {
			return err
		}

		// Collect erc20 events
		es, err := m.erc20Events(blockNumber, receipt.TxHash, r.Logs)
		if err != nil {
			return err
		}
		events = append(events, es...)
		totalTxsFee.Add(totalTxsFee, new(big.Int).Mul(txPrices[receipt.TxHash], new(big.Int).SetUint64(receipt.GasUsed)))
	}

	// Insert blocks
	minerBaseReward, uncleInclusionReward, uncleCBs, unclesReward, unclesHash := common.AccumulateRewards(block.Header(), block.Uncles())
	h, err := common.Header(block).AddReward(totalTxsFee, minerBaseReward, uncleInclusionReward, unclesReward, uncleCBs, unclesHash)
	if err != nil {
		return err
	}
	err = headerStore.Insert(ctx, h)
	if err != nil {
		return err
	}

	// Insert uncles
	for i, u := range block.Uncles() {
		// Insert a transfer event to represent uncle reward
		events = append(events, &model.Transfer{
			Address:     model.ETHBytes,
			BlockNumber: block.Number().Int64(),
			TxHash:      u.Hash().Bytes(),
			From:        model.RewardToUncle.Bytes(),
			To:          u.Coinbase.Bytes(),
			Value:       unclesReward[i].String(),
		})
	}

	// Insert a transfer event to represent miner reward
	events = append(events, &model.Transfer{
		Address:     model.ETHBytes,
		BlockNumber: block.Number().Int64(),
		TxHash:      block.Hash().Bytes(),
		From:        model.RewardToMiner.Bytes(),
		To:          block.Coinbase().Bytes(),
		Value:       h.MinerReward,
	})

	err = newTransferProcessor(block, m.tokenList, receipts, txs, subsStore, accountStore, balancer).process(ctx, events)
	if err != nil {
		return err
	}

	// Init new erc20 tokens if existed
	newTokens, err := m.initNewERC20(ctx, balancer, accountStore, subsStore, block)
	if err != nil {
		return err
	}
	for _, token := range newTokens {
		m.tokenList[gethCommon.BytesToAddress(token.Address)] = token
	}

	return nil
}

// delete deletes block and state data inside a DB transaction
func (m *manager) delete(ctx context.Context, dbTx DbOrTx, from, to int64) (err error) {
	headerStore := block_header.NewWithDB(dbTx, block_header.Cache())
	txStore := transaction.NewWithDB(dbTx)
	receiptStore := transaction_receipt.NewWithDB(dbTx)
	accountStore := account.NewWithDB(dbTx)
	subscriptionStore := subscription.NewWithDB(dbTx)

	err = headerStore.Delete(ctx, from, to)
	if err != nil {
		return
	}
	err = txStore.Delete(ctx, from, to)
	if err != nil {
		return
	}
	err = receiptStore.Delete(ctx, from, to)
	if err != nil {
		return
	}

	err = subscriptionStore.Reset(ctx, from, to)
	if err != nil {
		return
	}

	for addr, token := range m.tokenList {
		// Delete erc20 balances
		err = accountStore.DeleteAccounts(ctx, addr, from, to)
		if err != nil {
			return
		}

		// Delete erc20 events
		err = accountStore.DeleteTransfer(ctx, addr, from, to)
		if err != nil {
			return
		}

		if from <= token.BlockNumber && token.BlockNumber <= to {
			// If `from` is equal to the init block number of the contract, we need to remove data at `token.BlockNumber - 1` block. It's because we add extra data at that block once we handled a new erc20 contract.
			if from == token.BlockNumber {
				number := token.BlockNumber - 1
				err = accountStore.DeleteAccounts(ctx, addr, number, number)
				if err != nil {
					return
				}
			}
			// Reset token
			err = accountStore.BatchUpdateERC20BlockNumber(ctx, 0, [][]byte{
				addr.Bytes(),
			})
			if err != nil {
				return
			}
			// Remove token if it's initialized between `from` and `to`
			delete(m.tokenList, addr)
		}
	}
	return
}
