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

package account

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/jinzhu/gorm"
	"github.com/maichain/eth-indexer/model"
)

const (
	NameStateBlocks = "state_blocks"
	NameERC20       = "erc20"
	NameAccounts    = "accounts"
)

//go:generate mockery -name Store

type Store interface {
	// ERC 20
	InsertERC20(code *model.ERC20) error
	FindERC20(address common.Address) (result *model.ERC20, err error)
	ListERC20() ([]model.ERC20, error)

	// ERC 20 storage
	InsertERC20Storage(storage *model.ERC20Storage) error
	FindERC20Storage(address common.Address, key common.Hash, blockNr int64) (result *model.ERC20Storage, err error)
	DeleteERC20Storage(address common.Address, fromBlock int64) error

	// Accounts
	InsertAccount(account *model.Account) error
	FindAccount(address common.Address, blockNr ...int64) (result *model.Account, err error)
	DeleteAccounts(fromBlock int64) error

	// State block
	InsertStateBlock(block *model.StateBlock) error
	FindStateBlock(blockNr int64) (result *model.StateBlock, err error)
	DeleteStateBlocks(fromBlock int64) error
	LastStateBlock() (result *model.StateBlock, err error)
}

type store struct {
	db *gorm.DB
}

func NewWithDB(db *gorm.DB) Store {
	return &store{
		db: db,
	}
}

func (t *store) InsertERC20(code *model.ERC20) error {
	// Insert contract code
	if err := t.db.Table(NameERC20).Create(code).Error; err != nil {
		return err
	}
	// Create a table for this contract
	return t.db.CreateTable(model.ERC20Storage{
		Address: code.Address,
	}).Error
}

func (t *store) InsertERC20Storage(storage *model.ERC20Storage) error {
	return t.db.Table(storage.TableName()).Create(storage).Error
}

func (t *store) InsertAccount(account *model.Account) error {
	return t.db.Table(NameAccounts).Create(account).Error
}

func (t *store) InsertStateBlock(block *model.StateBlock) error {
	return t.db.Table(NameStateBlocks).Create(block).Error
}

func (t *store) LastStateBlock() (result *model.StateBlock, err error) {
	result = &model.StateBlock{}
	err = t.db.Table(NameStateBlocks).Order("number DESC").Limit(1).Find(result).Error
	return
}

func (t *store) DeleteAccounts(fromBlock int64) error {
	return t.db.Table(NameAccounts).Delete(model.Account{}, "block_number >= ?", fromBlock).Error
}

func (t *store) DeleteStateBlocks(fromBlock int64) error {
	return t.db.Table(NameStateBlocks).Delete(model.StateBlock{}, "number >= ?", fromBlock).Error
}

func (t *store) FindAccount(address common.Address, blockNr ...int64) (result *model.Account, err error) {
	result = &model.Account{}
	if len(blockNr) == 0 {
		err = t.db.Table(NameAccounts).Where("BINARY address = ?", address.Bytes()).Order("block_number DESC").Limit(1).Find(result).Error
	} else {
		err = t.db.Table(NameAccounts).Where("BINARY address = ? AND block_number <= ?", address.Bytes(), blockNr[0]).Order("block_number DESC").Limit(1).Find(result).Error
	}
	return
}

func (t *store) FindERC20(address common.Address) (result *model.ERC20, err error) {
	result = &model.ERC20{}
	err = t.db.Table(NameERC20).Where("BINARY address = ?", address.Bytes()).Limit(1).Find(result).Error
	return
}

func (t *store) ListERC20() (results []model.ERC20, err error) {
	results = []model.ERC20{}
	err = t.db.Table(NameERC20).Find(&results).Error
	return
}

func (t *store) FindERC20Storage(address common.Address, key common.Hash, blockNr int64) (result *model.ERC20Storage, err error) {
	result = &model.ERC20Storage{}
	err = t.db.Table(model.ERC20Storage{
		Address: address.Bytes(),
	}.TableName()).Where("BINARY key_hash = ? AND block_number <= ?", key.Bytes(), blockNr).Order("block_number DESC").Limit(1).Find(result).Error
	result.Address = address.Bytes()
	return
}

func (t *store) DeleteERC20Storage(address common.Address, fromBlock int64) error {
	return t.db.Table(model.ERC20Storage{
		Address: address.Bytes(),
	}.TableName()).Delete(model.ERC20Storage{}, "number >= ?", fromBlock).Error
}

func (t *store) FindStateBlock(blockNr int64) (result *model.StateBlock, err error) {
	result = &model.StateBlock{}
	err = t.db.Table(NameStateBlocks).Where("number <= ?", blockNr).Order("number DESC").Limit(1).Find(result).Error
	return
}
