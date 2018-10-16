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

package transaction_receipt

import (
	"context"
	"errors"
	"fmt"

	"github.com/getamis/eth-indexer/model"
	. "github.com/getamis/eth-indexer/store/sqldb"
	"github.com/jmoiron/sqlx"
)

//go:generate mockery -name Store
type Store interface {
	Insert(ctx context.Context, data *model.Receipt) error
	Delete(ctx context.Context, from, to int64) (err error)
	FindReceipt(ctx context.Context, hash []byte) (result *model.Receipt, err error)
}

const (
	insertReceiptSQL = "INSERT INTO `transaction_receipts` (`root`, `status`, `cumulative_gas_used`, `bloom`, `tx_hash`, `contract_address`, `gas_used`, `block_number`) VALUES (X'%s', %d, %d, X'%s', X'%s', X'%s', %d, %d)"
	insertLogSQL     = "INSERT INTO `receipt_logs` (`tx_hash`, `block_number`, `contract_address`, `event_name`, `topic1`, `topic2`, `topic3`, `data`) VALUES (X'%s', %d, X'%s', X'%s', X'%s', X'%s', X'%s', X'%s')"
	deleteReceiptSQL = "DELETE FROM `transaction_receipts` WHERE `block_number` >= %d AND `block_number` <= %d"
	deleteLogSQL     = "DELETE FROM `receipt_logs` WHERE `block_number` >= %d AND `block_number` <= %d"
	findReceiptSQL   = "SELECT * FROM `transaction_receipts` WHERE `tx_hash` = X'%s'"
	findLogSQL       = "SELECT * FROM `receipt_logs` WHERE `tx_hash` = X'%s'"
)

type store struct {
	db DbOrTx
}

func NewWithDB(db DbOrTx) Store {
	return &store{
		db: db,
	}
}

func (r *store) Insert(ctx context.Context, data *model.Receipt) (err error) {
	// Ensure we are in a db transaction
	var dbTx *sqlx.Tx
	db, ok := r.db.(*sqlx.DB)
	if ok {
		dbTx, err = db.BeginTxx(ctx, nil)
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
	} else {
		dbTx, ok = r.db.(*sqlx.Tx)
		if !ok {
			return errors.New("not in a transaction")
		}
	}

	// Insert receipt
	_, err = dbTx.ExecContext(ctx, fmt.Sprintf(insertReceiptSQL, Hex(data.Root), data.Status, data.CumulativeGasUsed, Hex(data.Bloom), Hex(data.TxHash), Hex(data.ContractAddress), data.GasUsed, data.BlockNumber))
	if err != nil {
		return err
	}
	// Insert logs
	for _, l := range data.Logs {
		_, err = dbTx.ExecContext(ctx, fmt.Sprintf(insertLogSQL, Hex(l.TxHash), data.BlockNumber, Hex(l.ContractAddress), Hex(l.EventName), Hex(l.Topic1), Hex(l.Topic2), Hex(l.Topic3), Hex(l.Data)))
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *store) Delete(ctx context.Context, from, to int64) (err error) {
	// Ensure we are in a db transaction
	var dbTx *sqlx.Tx
	db, ok := r.db.(*sqlx.DB)
	if ok {
		dbTx, err = db.BeginTxx(ctx, nil)
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
	} else {
		dbTx, ok = r.db.(*sqlx.Tx)
		if !ok {
			return errors.New("not in a transaction")
		}
	}

	// Delete receipt
	_, err = dbTx.ExecContext(ctx, fmt.Sprintf(deleteReceiptSQL, from, to))
	if err != nil {
		return err
	}
	// Delete logs
	_, err = dbTx.ExecContext(ctx, fmt.Sprintf(deleteLogSQL, from, to))
	if err != nil {
		return err
	}
	return nil
}

func (r *store) FindReceipt(ctx context.Context, hash []byte) (result *model.Receipt, err error) {
	// Ensure we are in a db transaction
	var dbTx *sqlx.Tx
	db, ok := r.db.(*sqlx.DB)
	if ok {
		dbTx, err = db.BeginTxx(ctx, nil)
		if err != nil {
			return nil, err
		}
		defer func() {
			if err != nil {
				dbTx.Rollback()
				return
			}
			err = dbTx.Commit()
		}()
	} else {
		dbTx, ok = r.db.(*sqlx.Tx)
		if !ok {
			return nil, errors.New("not in a transaction")
		}
	}

	// Find receipt
	receipt := &model.Receipt{}
	err = dbTx.GetContext(ctx, receipt, fmt.Sprintf(findReceiptSQL, Hex(hash)))
	if err != nil {
		return nil, err
	}

	// Find logs
	logs := []*model.Log{}
	err = dbTx.SelectContext(ctx, &logs, fmt.Sprintf(findLogSQL, Hex(hash)))
	if err != nil {
		return nil, err
	}
	receipt.Logs = logs
	return receipt, nil
}
