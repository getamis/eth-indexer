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
	"github.com/jinzhu/gorm"
	"github.com/maichain/eth-indexer/store/model"
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
