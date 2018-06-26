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
	"github.com/ethereum/go-ethereum/common"
	"github.com/getamis/eth-indexer/model"
	"github.com/jinzhu/gorm"
)

//go:generate mockery -name Store

type Store interface {
	// ERC 20
	InsertERC20(code *model.ERC20, onlySubscribe bool) error
	FindERC20(address common.Address) (result *model.ERC20, err error)
	ListERC20() ([]*model.ERC20, error)

	// ERC 20 storage
	InsertERC20Storage(storage *model.ERC20Storage) error
	FindERC20Storage(address common.Address, key common.Hash, blockNr int64) (result *model.ERC20Storage, err error)
	DeleteERC20Storage(address common.Address, from, to int64) error
	LastSyncERC20Storage(address common.Address, blockNr int64) (result int64, err error)

	// Accounts
	InsertAccount(account *model.Account) error
	FindAccount(contractAddress common.Address, address common.Address, blockNr ...int64) (result *model.Account, err error)
	DeleteAccounts(contractAddress common.Address, from, to int64) error

	// Transfer events
	InsertTransfer(event *model.Transfer) error
	FindTransfer(contractAddress common.Address, address common.Address, blockNr ...int64) (result *model.Transfer, err error)
	DeleteTransfer(contractAddress common.Address, from, to int64) error
}

type store struct {
	db *gorm.DB
}

func NewWithDB(db *gorm.DB) Store {
	return &store{
		db: db,
	}
}

func (t *store) InsertERC20(code *model.ERC20, onlySubscribe bool) error {
	// Insert contract code
	if err := t.db.Create(code).Error; err != nil {
		return err
	}
	if onlySubscribe {
		// Create a account table for this contract
		if err := t.db.CreateTable(model.Account{
			ContractAddress: code.Address,
		}).Error; err != nil {
			return err
		}
	} else {
		// Create a table for this contract
		if err := t.db.CreateTable(model.ERC20Storage{
			Address: code.Address,
		}).Error; err != nil {
			return err
		}
	}
	// Create erc20 transfer event table
	if err := t.db.CreateTable(model.Transfer{
		Address: code.Address,
	}).Error; err != nil {
		return err
	}
	return nil
}

func (t *store) FindERC20(address common.Address) (result *model.ERC20, err error) {
	result = &model.ERC20{}
	err = t.db.Where("address = ?", address.Bytes()).Limit(1).Find(result).Error
	return
}

func (t *store) ListERC20() (results []*model.ERC20, err error) {
	results = []*model.ERC20{}
	err = t.db.Find(&results).Error
	return
}

func (t *store) InsertERC20Storage(storage *model.ERC20Storage) error {
	return t.db.Table(storage.TableName()).Create(storage).Error
}

func (t *store) FindERC20Storage(contractAddress common.Address, key common.Hash, blockNr int64) (result *model.ERC20Storage, err error) {
	result = &model.ERC20Storage{}
	err = t.db.Table(model.ERC20Storage{
		Address: contractAddress.Bytes(),
	}.TableName()).Where("key_hash = ? AND block_number <= ?", key.Bytes(), blockNr).Order("block_number DESC").Limit(1).Find(result).Error
	result.Address = contractAddress.Bytes()
	return
}

func (t *store) DeleteERC20Storage(contractAddress common.Address, from, to int64) error {
	return t.db.Table(model.ERC20Storage{
		Address: contractAddress.Bytes(),
	}.TableName()).Delete(model.ERC20Storage{}, "block_number >= ? AND block_number <= ?", from, to).Error
}

func (t *store) LastSyncERC20Storage(address common.Address, blockNr int64) (int64, error) {
	result := &model.ERC20Storage{}
	err := t.db.Table(model.ERC20Storage{
		Address: address.Bytes(),
	}.TableName()).Where("block_number <= ?", blockNr).Order("block_number DESC").Limit(1).Find(result).Error
	if err != nil {
		return 0, err
	}
	return result.BlockNumber, nil
}

func (t *store) InsertAccount(account *model.Account) error {
	return t.db.Create(account).Error
}

func (t *store) FindAccount(contractAddress common.Address, address common.Address, blockNr ...int64) (result *model.Account, err error) {
	result = &model.Account{
		ContractAddress: contractAddress.Bytes(),
	}
	if len(blockNr) == 0 {
		err = t.db.Where("address = ?", address.Bytes()).Order("block_number DESC").Limit(1).Find(result).Error
	} else {
		err = t.db.Where("address = ? AND block_number <= ?", address.Bytes(), blockNr[0]).Order("block_number DESC").Limit(1).Find(result).Error
	}
	return
}

func (t *store) DeleteAccounts(contractAddress common.Address, from, to int64) error {
	return t.db.Delete(model.Account{
		ContractAddress: contractAddress.Bytes(),
	}, "block_number >= ? AND block_number <= ?", from, to).Error
}

func (t *store) InsertTransfer(event *model.Transfer) error {
	return t.db.Create(event).Error
}

func (t *store) FindTransfer(contractAddress common.Address, address common.Address, blockNr ...int64) (result *model.Transfer, err error) {
	result = &model.Transfer{
		Address: contractAddress.Bytes(),
	}
	if len(blockNr) == 0 {
		err = t.db.Where("address = ?", address.Bytes()).Order("block_number DESC").Limit(1).Find(result).Error
	} else {
		err = t.db.Where("address = ? AND block_number <= ?", address.Bytes(), blockNr[0]).Order("block_number DESC").Limit(1).Find(result).Error
	}
	return
}

func (t *store) DeleteTransfer(contractAddress common.Address, from, to int64) error {
	return t.db.Delete(model.Transfer{
		Address: contractAddress.Bytes(),
	}, "block_number >= ? AND block_number <= ?", from, to).Error
}
