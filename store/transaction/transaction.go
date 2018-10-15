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

package transaction

import (
	"context"
	"fmt"

	"github.com/getamis/eth-indexer/model"
	. "github.com/getamis/eth-indexer/store/sqldb"
)

//go:generate mockery -name Store
type Store interface {
	Insert(ctx context.Context, data *model.Transaction) error
	Delete(ctx context.Context, from, to int64) (err error)
	FindTransaction(ctx context.Context, hash []byte) (result *model.Transaction, err error)
	FindTransactionsByBlockHash(ctx context.Context, blockHash []byte) (result []*model.Transaction, err error)
}

const (
	insertSQL                     = "INSERT INTO transactions (`hash`, block_hash, `from`, `to`, nonce, gas_price, gas_limit, amount, payload, block_number) VALUES (X'%s', X'%s', X'%s', X'%s', %d, %d, %d, '%s', X'%s', %d)"
	deleteSQL                     = "DELETE FROM transactions WHERE block_number >= %d AND block_number <= %d"
	findTransactionSQL            = "SELECT * FROM transactions WHERE `hash` = X'%s'"
	findTransactionByBlockHashSQL = "SELECT * FROM transactions WHERE `block_hash` = X'%s'"
)

type store struct {
	db DbOrTx
}

func NewWithDB(db DbOrTx) Store {
	return &store{
		db: db,
	}
}

func (t *store) Insert(ctx context.Context, data *model.Transaction) error {
	_, err := t.db.ExecContext(ctx, fmt.Sprintf(insertSQL, Hex(data.Hash), Hex(data.BlockHash), Hex(data.From), Hex(data.To), data.Nonce, data.GasPrice, data.GasLimit, data.Amount, Hex(data.Payload), data.BlockNumber))
	return err
}

func (t *store) Delete(ctx context.Context, from, to int64) error {
	_, err := t.db.ExecContext(ctx, fmt.Sprintf(deleteSQL, from, to))
	return err
}

func (t *store) FindTransaction(ctx context.Context, hash []byte) (*model.Transaction, error) {
	result := &model.Transaction{}
	err := t.db.GetContext(ctx, result, fmt.Sprintf(findTransactionSQL, Hex(hash)))
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (t *store) FindTransactionsByBlockHash(ctx context.Context, blockHash []byte) ([]*model.Transaction, error) {
	result := []*model.Transaction{}
	err := t.db.SelectContext(ctx, &result, fmt.Sprintf(findTransactionByBlockHashSQL, Hex(blockHash)))
	if err != nil {
		return nil, err
	}
	return result, nil
}
