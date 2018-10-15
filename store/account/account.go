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

package account

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/getamis/eth-indexer/model"
	. "github.com/getamis/eth-indexer/store/sqldb"
	"github.com/jmoiron/sqlx"
)

//go:generate mockery -name Store

type Store interface {
	// ERC 20
	InsertERC20(ctx context.Context, code *model.ERC20) error
	BatchUpdateERC20BlockNumber(ctx context.Context, blockNumber int64, addrs [][]byte) error
	FindERC20(ctx context.Context, address common.Address) (result *model.ERC20, err error)
	ListERC20(ctx context.Context) ([]*model.ERC20, error)
	ListOldERC20(ctx context.Context) ([]*model.ERC20, error)
	ListNewERC20(ctx context.Context) ([]*model.ERC20, error)

	// Accounts
	InsertAccount(ctx context.Context, account *model.Account) error
	FindAccount(ctx context.Context, contractAddress common.Address, address common.Address, blockNr ...int64) (result *model.Account, err error)
	FindLatestAccounts(ctx context.Context, contractAddress common.Address, addrs [][]byte) (result []*model.Account, err error)
	DeleteAccounts(ctx context.Context, contractAddress common.Address, from, to int64) error

	// Transfer events
	InsertTransfer(ctx context.Context, event *model.Transfer) error
	FindTransfer(ctx context.Context, contractAddress common.Address, address common.Address, blockNr ...int64) (result *model.Transfer, err error)
	FindAllTransfers(ctx context.Context, contractAddress common.Address, address common.Address) (result []*model.Transfer, err error)
	DeleteTransfer(ctx context.Context, contractAddress common.Address, from, to int64) error
}

const (
	insertERC20SQL                 = "INSERT INTO erc20 (block_number, address, total_supply, decimals, name) VALUES (%d, X'%s', '%s', %d, '%s')"
	createERC20TableSQL            = "CREATE TABLE `%s`(`block_number` bigint(20) DEFAULT NULL, `address` varbinary(20) DEFAULT NULL, `balance` varchar(32) DEFAULT NULL, UNIQUE INDEX `idx_block_number_address` (`block_number`,`address`), INDEX `block_number` (`block_number`), INDEX `address` (`address`))"
	createERC20TransferSQL         = "CREATE TABLE `%s` (`block_number` bigint(20) DEFAULT NULL,`tx_hash` varbinary(32) DEFAULT NULL, `from` varbinary(20) DEFAULT NULL, `to` varbinary(20) DEFAULT NULL, `value` varchar(32) DEFAULT NULL, INDEX `block_number` (`block_number`), INDEX `tx_hash` (`tx_hash`), INDEX `from` (`from`), INDEX `to` (`to`))"
	batchUpdateERC20BlockNumberSQL = "UPDATE erc20 SET block_number = %d WHERE address IN (%s)"
	findERC20SQL                   = "SELECT * FROM erc20 WHERE address = X'%s'"
	listERC20SQL                   = "SELECT * FROM erc20"
	listOldERC20SQL                = "SELECT * FROM erc20 WHERE block_number > 0"
	listNewERC20SQL                = "SELECT * FROM erc20 WHERE block_number = 0"
	insertAccountSQL               = "INSERT INTO `%s` (block_number, address, balance) VALUES (%d, X'%s', '%s')"
	findAccountSQL                 = "SELECT * FROM `%s` WHERE address = X'%s' ORDER BY block_number DESC"
	findAccountByNumberSQL         = "SELECT * FROM `%s` WHERE address = X'%s' AND block_number <= %d ORDER BY block_number DESC"
	deleteAccountsSQL              = "DELETE FROM `%s` WHERE block_number >= %d AND block_number <= %d"
	insertTransferSQL              = "INSERT INTO `%s` (block_number, tx_hash, `from`, `to`, value) VALUES (%d, X'%s', X'%s', X'%s', '%s')"
	findTransferSQL                = "SELECT * FROM `%s` WHERE (`from` = X'%s' OR `to` = X'%s') ORDER BY block_number DESC"
	findTransferByNumberSQL        = "SELECT * FROM `%s` WHERE (`from` = X'%s' OR `to` = X'%s') AND block_number <= %d ORDER BY block_number DESC"
	deleteTransferSQL              = "DELETE FROM `%s` WHERE block_number >= %d AND block_number <= %d"
)

type store struct {
	db DbOrTx
}

func NewWithDB(db DbOrTx) Store {
	return &store{
		db: db,
	}
}

func (t *store) InsertERC20(ctx context.Context, code *model.ERC20) (err error) {
	// Ensure we are in a db transaction
	var dbTx *sqlx.Tx
	db, ok := t.db.(*sqlx.DB)
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
		dbTx, ok = t.db.(*sqlx.Tx)
		if !ok {
			return errors.New("not in a transaction")
		}
	}

	// Insert contract code
	_, err = dbTx.ExecContext(ctx, fmt.Sprintf(insertERC20SQL, code.BlockNumber, Hex(code.Address), code.TotalSupply, code.Decimals, code.Name))
	if err != nil {
		return err
	}

	// Create a account table for this contract
	_, err = dbTx.ExecContext(ctx, fmt.Sprintf(createERC20TableSQL, model.Account{
		ContractAddress: code.Address,
	}.TableName()))
	if err != nil {
		return err
	}

	// Create erc20 transfer event table
	_, err = dbTx.ExecContext(ctx, fmt.Sprintf(createERC20TransferSQL, model.Transfer{
		Address: code.Address,
	}.TableName()))
	return err
}

func (t *store) FindERC20(ctx context.Context, address common.Address) (*model.ERC20, error) {
	result := &model.ERC20{}
	err := t.db.GetContext(ctx, result, fmt.Sprintf(findERC20SQL, Hex(address.Bytes())))
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (t *store) ListERC20(ctx context.Context) ([]*model.ERC20, error) {
	result := []*model.ERC20{}
	err := t.db.SelectContext(ctx, &result, listERC20SQL)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (t *store) ListOldERC20(ctx context.Context) ([]*model.ERC20, error) {
	result := []*model.ERC20{}
	err := t.db.SelectContext(ctx, &result, listOldERC20SQL)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (t *store) ListNewERC20(ctx context.Context) ([]*model.ERC20, error) {
	result := []*model.ERC20{}
	err := t.db.SelectContext(ctx, &result, listNewERC20SQL)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (t *store) BatchUpdateERC20BlockNumber(ctx context.Context, blockNumber int64, addrs [][]byte) error {
	if len(addrs) == 0 {
		return nil
	}

	_, err := t.db.ExecContext(ctx, fmt.Sprintf(batchUpdateERC20BlockNumberSQL, blockNumber, InClauseForBytes(addrs)))
	return err
}

func (t *store) InsertAccount(ctx context.Context, account *model.Account) error {
	_, err := t.db.ExecContext(ctx, fmt.Sprintf(insertAccountSQL, account.TableName(), account.BlockNumber, Hex(account.Address), account.Balance))
	return err
}

func (t *store) FindAccount(ctx context.Context, contractAddress common.Address, address common.Address, blockNr ...int64) (result *model.Account, err error) {
	result = &model.Account{
		ContractAddress: contractAddress.Bytes(),
	}
	if len(blockNr) == 0 {
		err = t.db.GetContext(ctx, result, fmt.Sprintf(findAccountSQL, result.TableName(), Hex(address.Bytes())))
	} else {
		err = t.db.GetContext(ctx, result, fmt.Sprintf(findAccountByNumberSQL, result.TableName(), Hex(address.Bytes()), blockNr[0]))
	}
	return
}

func (t *store) FindLatestAccounts(ctx context.Context, contractAddress common.Address, addrs [][]byte) (result []*model.Account, err error) {
	if len(addrs) == 0 {
		return []*model.Account{}, nil
	}

	result = []*model.Account{}
	acct := model.Account{
		ContractAddress: contractAddress.Bytes(),
	}
	// The following query does not work because the select fields needs to also be in group by fields (ONLY_FULL_GROUP_BY mode)
	// "select address, balance, MAX(block_number) as block_number from %s where address in (?) group by address"
	// and the following query
	// "select address, balance, MAX(block_number) as block_number from %s where address in (?) group by (address, balance)"
	// is not what we want, because (address, balance) isn't unique
	query := fmt.Sprintf(
		"select t1.address, t1.block_number, t1.balance from %s as t1, (select address, MAX(block_number) as block_number from %s where address in (%s) group by address) as t2 where t1.address = t2.address and t1.block_number = t2.block_number",
		acct.TableName(), acct.TableName(), InClauseForBytes(addrs))
	err = t.db.SelectContext(ctx, &result, query)
	if err != nil {
		return
	}
	return
}

func (t *store) DeleteAccounts(ctx context.Context, contractAddress common.Address, from, to int64) error {
	_, err := t.db.ExecContext(ctx, fmt.Sprintf(deleteAccountsSQL, model.Account{
		ContractAddress: contractAddress.Bytes(),
	}.TableName(), from, to))
	return err
}

func (t *store) InsertTransfer(ctx context.Context, event *model.Transfer) error {
	_, err := t.db.ExecContext(ctx, fmt.Sprintf(insertTransferSQL, event.TableName(), event.BlockNumber, Hex(event.TxHash), Hex(event.From), Hex(event.To), event.Value))
	return err
}

func (t *store) FindTransfer(ctx context.Context, contractAddress common.Address, address common.Address, blockNr ...int64) (*model.Transfer, error) {
	result := &model.Transfer{
		Address: contractAddress.Bytes(),
	}
	var err error
	if len(blockNr) == 0 {
		err = t.db.GetContext(ctx, result, fmt.Sprintf(findTransferSQL, result.TableName(), Hex(address.Bytes()), Hex(address.Bytes())))
	} else {
		err = t.db.GetContext(ctx, result, fmt.Sprintf(findTransferByNumberSQL, result.TableName(), Hex(address.Bytes()), Hex(address.Bytes()), blockNr[0]))
	}
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (t *store) FindAllTransfers(ctx context.Context, contractAddress common.Address, address common.Address) ([]*model.Transfer, error) {
	tableName := model.Transfer{
		Address: contractAddress.Bytes(),
	}.TableName()

	result := []*model.Transfer{}
	err := t.db.SelectContext(ctx, &result, fmt.Sprintf(findTransferSQL, tableName, Hex(address.Bytes()), Hex(address.Bytes())))
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (t *store) DeleteTransfer(ctx context.Context, contractAddress common.Address, from, to int64) error {
	_, err := t.db.ExecContext(ctx, fmt.Sprintf(deleteTransferSQL, model.Transfer{
		Address: contractAddress.Bytes(),
	}.TableName(), from, to))
	return err
}
