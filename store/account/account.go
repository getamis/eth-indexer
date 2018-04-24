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
	NameStateBlocks  = "state_blocks"
	NameContractCode = "contract_code"
	NameContracts    = "contracts"
	NameAccounts     = "accounts"
)

type Store interface {
	InsertContractCode(code model.ContractCode) error
	InsertContract(contract model.Contract) error
	InsertAccount(account model.Account) error
	InsertStateBlock(block model.StateBlock) error
	LastStateBlock() (result *model.StateBlock, err error)

	FindAccount(address common.Address, blockNr ...int64) (result *model.Account, err error)
	FindContract(address common.Address, blockNr ...int64) (result *model.Contract, err error)
	FindContractCode(address common.Address) (result *model.ContractCode, err error)
	FindStateBlock(address common.Address, blockNr ...int64) (result *model.StateBlock, err error)
}

type store struct {
	db *gorm.DB
}

func NewWithDB(db *gorm.DB) Store {
	return &store{
		db: db,
	}
}

func (t *store) InsertContractCode(code model.ContractCode) error {
	return t.db.Table(NameContractCode).Create(code).Error
}

func (t *store) InsertContract(contract model.Contract) error {
	return t.db.Table(NameContracts).Create(contract).Error
}

func (t *store) InsertAccount(account model.Account) error {
	return t.db.Table(NameAccounts).Create(account).Error
}

func (t *store) InsertStateBlock(block model.StateBlock) error {
	return t.db.Table(NameStateBlocks).Create(block).Error
}

func (t *store) LastStateBlock() (result *model.StateBlock, err error) {
	result = &model.StateBlock{}
	err = t.db.Table(NameStateBlocks).Order("number DESC").Limit(1).Find(result).Error
	return
}

func (t *store) FindAccount(address common.Address, blockNr ...int64) (result *model.Account, err error) {
	result = &model.Account{}
	if len(blockNr) == 0 {
		err = t.db.Table(NameAccounts).Where(&model.Account{
			Address: address.Bytes(),
		}).Order("block_number DESC").Limit(1).Find(result).Error
	} else {
		err = t.db.Table(NameAccounts).Where("address = ? AND block_number <= ?", address.Bytes(), blockNr[0]).Order("block_number DESC").Limit(1).Find(result).Error
	}
	return
}

func (t *store) FindContract(address common.Address, blockNr ...int64) (result *model.Contract, err error) {
	result = &model.Contract{}
	if len(blockNr) == 0 {
		err = t.db.Table(NameContracts).Where(&model.Contract{
			Address: address.Bytes(),
		}).Order("block_number DESC").Limit(1).Find(result).Error
	} else {
		err = t.db.Table(NameContracts).Where("address = ? AND block_number <= ?", address.Bytes(), blockNr[0]).Order("block_number DESC").Limit(1).Find(result).Error
	}
	return
}

func (t *store) FindContractCode(address common.Address) (result *model.ContractCode, err error) {
	result = &model.ContractCode{}
	err = t.db.Table(NameContracts).Where(&model.ContractCode{
		Address: address.Bytes(),
	}).Limit(1).Find(result).Error
	return
}

func (t *store) FindStateBlock(address common.Address, blockNr ...int64) (result *model.StateBlock, err error) {
	result = &model.StateBlock{}
	if len(blockNr) == 0 {
		err = t.db.Table(NameStateBlocks).Order("number DESC").Limit(1).Find(result).Error
	} else {
		err = t.db.Table(NameStateBlocks).Where("number <= ?", blockNr[0]).Order("number DESC").Limit(1).Find(result).Error
	}
	return
}
