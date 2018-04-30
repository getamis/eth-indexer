// Copyright © 2018 AMIS Technologies
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package account

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"

	ethCommon "github.com/ethereum/go-ethereum/common"
	gethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/getamis/sirius/log"
	"github.com/maichain/eth-indexer/common"
	"github.com/maichain/eth-indexer/model"
)

var (
	// ErrNotSelf retruns if the address is not my contraact address
	ErrNotSelf = errors.New("not self address")
)

// Implement vm.ContractRef
type account struct {
	address ethCommon.Address
}

func (account *account) ReturnGas(*big.Int, *big.Int)                                 {}
func (account *account) Address() ethCommon.Address                                   { return account.address }
func (account *account) Value() *big.Int                                              { return ethCommon.Big0 }
func (account *account) SetCode(ethCommon.Hash, []byte)                               {}
func (account *account) ForEachStorage(callback func(key, value ethCommon.Hash) bool) {}

// Implement vm.StateDB. In current version, we only read the states in the given account (contract).
type contractDB struct {
	code    *model.ContractCode
	account *model.Contract
	err     error
}

func (contractDB) CreateAccount(addr ethCommon.Address)                                        {}
func (contractDB) SubBalance(addr ethCommon.Address, balance *big.Int)                         {}
func (contractDB) AddBalance(addr ethCommon.Address, balance *big.Int)                         {}
func (contractDB) SetNonce(addr ethCommon.Address, nonce uint64)                               {}
func (contractDB) SetCode(addr ethCommon.Address, codes []byte)                                {}
func (contractDB) SetState(addr ethCommon.Address, hash1 ethCommon.Hash, hash2 ethCommon.Hash) {}
func (contractDB) Suicide(addr ethCommon.Address) bool                                         { return false }
func (contractDB) HasSuicided(addr ethCommon.Address) bool                                     { return false }
func (contractDB) RevertToSnapshot(snap int)                                                   {}
func (contractDB) Snapshot() int                                                               { return 0 }
func (contractDB) AddLog(*types.Log)                                                           {}
func (contractDB) AddPreimage(hash ethCommon.Hash, images []byte)                              {}
func (contractDB) ForEachStorage(addr ethCommon.Address, f func(ethCommon.Hash, ethCommon.Hash) bool) {
}
func (contractDB) AddRefund(fund uint64) {}
func (contractDB) GetRefund() uint64     { return 0 }

// self checks whether the address is the contract address.
func (db *contractDB) self(addr ethCommon.Address) (result bool) {
	return addr == ethCommon.BytesToAddress(db.account.Address)
}

// mustBeSelf checks whether the address is the contract address. If not, set error to ErrNotSelf
func (db *contractDB) mustBeSelf(addr ethCommon.Address) (result bool) {
	defer func() {
		if !result {
			db.err = ErrNotSelf
		}
	}()
	return db.self(addr)
}
func (db contractDB) Exist(addr ethCommon.Address) bool {
	return db.self(addr)
}
func (db contractDB) Empty(addr ethCommon.Address) bool {
	return !db.self(addr)
}
func (db contractDB) GetBalance(addr ethCommon.Address) *big.Int {
	if db.mustBeSelf(addr) {
		v, ok := new(big.Int).SetString(db.account.Balance, 10)
		if ok {
			return v
		}
		return ethCommon.Big0

	}
	return ethCommon.Big0
}
func (db contractDB) GetNonce(addr ethCommon.Address) uint64 {
	if db.mustBeSelf(addr) {
		return uint64(db.account.Nonce)
	}
	return 0
}
func (db contractDB) GetCodeHash(addr ethCommon.Address) ethCommon.Hash {
	if db.mustBeSelf(addr) {
		return ethCommon.BytesToHash(db.code.Hash)
	}
	return ethCommon.Hash{}
}
func (db contractDB) GetCode(addr ethCommon.Address) []byte {
	if db.mustBeSelf(addr) {
		return ethCommon.Hex2Bytes(db.code.Code)
	}
	return []byte{}
}
func (db contractDB) GetCodeSize(addr ethCommon.Address) int {
	if db.mustBeSelf(addr) {
		return len(db.GetCode(addr))
	}
	return 0
}
func (db contractDB) GetState(addr ethCommon.Address, hash ethCommon.Hash) ethCommon.Hash {
	if db.mustBeSelf(addr) {
		hashStr := ethCommon.Bytes2Hex(hash.Bytes())
		storage := make(map[string]string)
		err := json.Unmarshal(db.account.Storage, &storage)
		if err != nil {
			return ethCommon.Hash{}
		}

		s, ok := storage[hashStr]
		if ok {
			enc := ethCommon.Hex2Bytes(s)
			if len(enc) > 0 {
				_, content, _, _ := rlp.Split(enc)
				return ethCommon.BytesToHash(content)
			}
		}
	}
	return ethCommon.Hash{}
}

func (api *dbAPI) GetERC20Balance(ctx context.Context, contractAddress, address gethCommon.Address, blockNr int64) (*big.Int, *big.Int, error) {
	logger := log.New("contractAddr", contractAddress.Hex(), "addr", address.Hex(), "number", blockNr)
	// Find contract code
	contractCode, err := api.store.FindContractCode(contractAddress)
	if err != nil {
		logger.Error("Failed to find contract code", "err", err)
		return nil, nil, err
	}

	// Find state block
	var stateBlock *model.StateBlock
	if common.IsLatestBlock(blockNr) {
		stateBlock, err = api.store.LastStateBlock()
	} else {
		stateBlock, err = api.store.FindStateBlock(blockNr)
	}
	// State block should not have not found error
	if err != nil {
		logger.Error("Failed to find state block", "err", err)
		return nil, nil, err
	}
	blockNumber := big.NewInt(stateBlock.Number)

	// Find contract
	contract, err := api.store.FindContract(contractAddress, stateBlock.Number)
	if err != nil {
		logger.Error("Failed to find contract", "err", err)
		return nil, nil, err
	}

	// Get balance from contract
	db := &contractDB{
		code:    contractCode,
		account: contract,
	}
	balance, err := BalanceOf(db, contractAddress, address)
	if err != nil {
		logger.Error("Failed to get balance", "err", err)
		return nil, nil, err
	}
	if db.err != nil {
		logger.Error("Failed to get balance due to state db error", "err", db.err)
		return nil, nil, err
	}
	return balance, blockNumber, nil
}
