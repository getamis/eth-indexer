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

package block_header

import (
	"context"
	"fmt"
	"time"

	"github.com/getamis/eth-indexer/model"
	. "github.com/getamis/eth-indexer/store/sqldb"
)

//go:generate mockery -name Store

type Store interface {
	InsertTd(ctx context.Context, data *model.TotalDifficulty) error
	Insert(ctx context.Context, data *model.Header) error
	Delete(ctx context.Context, from, to int64) (err error)
	FindTd(ctx context.Context, hash []byte) (result *model.TotalDifficulty, err error)
	FindBlockByNumber(ctx context.Context, blockNumber int64) (result *model.Header, err error)
	FindBlockByHash(ctx context.Context, hash []byte) (result *model.Header, err error)
	FindLatestBlock(ctx context.Context) (result *model.Header, err error)
	CountBlocks(ctx context.Context) (uint64, error)
}

const (
	insertTdSQL          = "INSERT INTO `total_difficulty` (`block`, `hash`, `td`) VALUES (%d, X'%s', '%s')"
	insertSQL            = "INSERT INTO `block_headers` (`hash`, `parent_hash`, `uncle_hash`, `coinbase`, `root`, `tx_hash`, `receipt_hash`, `difficulty`, `number`, `gas_limit`, `gas_used`, `time`, `extra_data`, `mix_digest`, `nonce`, `miner_reward`, `uncles_inclusion_reward`, `txs_fee`, `uncle1_reward`, `uncle1_coinbase`, `uncle1_hash`, `uncle2_reward`, `uncle2_coinbase`, `uncle2_hash`, `created_at`) VALUES (X'%s', X'%s', X'%s', X'%s', X'%s', X'%s', X'%s', %d, %d, %d, %d, %d, X'%s', X'%s', X'%s', '%s', '%s', '%s', '%s', X'%s', X'%s', '%s', X'%s', X'%s', '%s')"
	deleteSQL            = "DELETE FROM `block_headers` WHERE `number` >= %d AND `number` <= %d"
	findTdSQL            = "SELECT * FROM `total_difficulty` WHERE `hash` = X'%s'"
	findBlockByNumberSQL = "SELECT * FROM `block_headers` WHERE `number` = %d"
	findBlockByHashSQL   = "SELECT * FROM `block_headers` WHERE `hash` = X'%s'"
	findLatestBlockSQL   = "SELECT * FROM `block_headers` ORDER BY `number` DESC LIMIT 1"
	countBlocksSQL       = "SELECT COUNT(*) FROM `block_headers`"
)

type store struct {
	db DbOrTx
}

func NewWithDB(db DbOrTx, opts ...Option) Store {
	var s Store = &store{
		db: db,
	}

	o := &Options{}
	for _, opt := range opts {
		opt(o)
	}

	for _, mw := range o.Middlewares {
		s = mw(s)
	}

	return s
}

func (t *store) InsertTd(ctx context.Context, data *model.TotalDifficulty) error {
	_, err := t.db.ExecContext(ctx, fmt.Sprintf(insertTdSQL, data.Block, Hex(data.Hash), data.Td))
	return err
}

func (t *store) Insert(ctx context.Context, data *model.Header) error {
	nowStr := ToTimeStr(time.Now())
	_, err := t.db.ExecContext(ctx, fmt.Sprintf(insertSQL, Hex(data.Hash), Hex(data.ParentHash), Hex(data.UncleHash), Hex(data.Coinbase), Hex(data.Root), Hex(data.TxHash), Hex(data.ReceiptHash), data.Difficulty, data.Number, data.GasLimit, data.GasUsed, data.Time, Hex(data.ExtraData), Hex(data.MixDigest), Hex(data.Nonce), data.MinerReward, data.UnclesInclusionReward, data.TxsFee, data.Uncle1Reward, Hex(data.Uncle1Coinbase), Hex(data.Uncle1Hash), data.Uncle2Reward, Hex(data.Uncle2Coinbase), Hex(data.Uncle2Hash), nowStr))
	return err
}

func (t *store) Delete(ctx context.Context, from, to int64) error {
	_, err := t.db.ExecContext(ctx, fmt.Sprintf(deleteSQL, from, to))
	return err
}

func (t *store) FindTd(ctx context.Context, hash []byte) (*model.TotalDifficulty, error) {
	result := &model.TotalDifficulty{}
	err := t.db.GetContext(ctx, result, fmt.Sprintf(findTdSQL, Hex(hash)))
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (t *store) FindBlockByNumber(ctx context.Context, blockNumber int64) (*model.Header, error) {
	result := &model.Header{}
	err := t.db.GetContext(ctx, result, fmt.Sprintf(findBlockByNumberSQL, blockNumber))
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (t *store) FindBlockByHash(ctx context.Context, hash []byte) (*model.Header, error) {
	result := &model.Header{}
	err := t.db.GetContext(ctx, result, fmt.Sprintf(findBlockByHashSQL, Hex(hash)))
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (t *store) FindLatestBlock(ctx context.Context) (*model.Header, error) {
	result := &model.Header{}
	err := t.db.GetContext(ctx, result, fmt.Sprintf(findLatestBlockSQL))
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (t *store) CountBlocks(ctx context.Context) (uint64, error) {
	var count uint64
	err := t.db.GetContext(ctx, &count, countBlocksSQL)
	if err != nil {
		return 0, err
	}
	return count, nil
}
