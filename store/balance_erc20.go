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
	"context"
	"errors"
	"math/big"

	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/getamis/eth-indexer/common"
	"github.com/getamis/eth-indexer/model"
	"github.com/getamis/eth-indexer/store/account"
	"github.com/getamis/sirius/log"
	"github.com/jinzhu/gorm"
	"github.com/shopspring/decimal"
)

var (
	// ErrNotSelf retruns if the address is not my contraact address
	ErrNotSelf = errors.New("not self address")
)

// Implement vm.ContractRef
type contractAccount struct {
	address ethCommon.Address
}

func (account *contractAccount) ReturnGas(*big.Int, *big.Int)                                 {}
func (account *contractAccount) Address() ethCommon.Address                                   { return account.address }
func (account *contractAccount) Value() *big.Int                                              { return ethCommon.Big0 }
func (account *contractAccount) SetCode(ethCommon.Hash, []byte)                               {}
func (account *contractAccount) ForEachStorage(callback func(key, value ethCommon.Hash) bool) {}

// Implement vm.StateDB. In current version, we only read the states in the given account (contract).
type contractDB struct {
	blockNumber  int64
	code         *model.ERC20
	account      *model.Account
	accountStore account.Store
	err          error
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
func (contractDB) AddRefund(fund uint64)             {}
func (contractDB) GetRefund() uint64                 { return 0 }
func (contractDB) AddTransferLog(*types.TransferLog) {}

// self checks whether the address is the contract address.
func (db *contractDB) self(addr ethCommon.Address) bool {
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
func (db *contractDB) GetBalance(addr ethCommon.Address) *big.Int {
	if db.mustBeSelf(addr) {
		v, ok := new(big.Int).SetString(db.account.Balance, 10)
		if ok {
			return v
		}
		return ethCommon.Big0

	}
	return ethCommon.Big0
}
func (db *contractDB) GetNonce(addr ethCommon.Address) uint64 {
	return 0
}
func (db *contractDB) GetCodeHash(addr ethCommon.Address) ethCommon.Hash {
	if db.mustBeSelf(addr) {
		return crypto.Keccak256Hash(db.code.Code)
	}
	return ethCommon.Hash{}
}
func (db *contractDB) GetCode(addr ethCommon.Address) []byte {
	if db.mustBeSelf(addr) {
		return db.code.Code
	}
	return []byte{}
}
func (db *contractDB) GetCodeSize(addr ethCommon.Address) int {
	if db.mustBeSelf(addr) {
		return len(db.GetCode(addr))
	}
	return 0
}
func (db *contractDB) GetState(addr ethCommon.Address, key ethCommon.Hash) ethCommon.Hash {
	if db.mustBeSelf(addr) {
		s, err := db.accountStore.FindERC20Storage(addr, key, db.blockNumber)
		if err != nil {
			// not found error means there is no storage at this block number
			if err != gorm.ErrRecordNotFound {
				db.err = err
			}
			return ethCommon.Hash{}
		}
		return ethCommon.BytesToHash(s.Value)
	}
	return ethCommon.Hash{}
}

func (srv *serviceManager) GetERC20Balance(ctx context.Context, contractAddress, address ethCommon.Address, blockNr int64) (*decimal.Decimal, *big.Int, error) {
	logger := log.New("contractAddr", contractAddress.Hex(), "addr", address.Hex(), "number", blockNr)
	// Find contract code
	erc20, err := srv.FindERC20(contractAddress)
	if err != nil {
		logger.Error("Failed to find contract code", "err", err)
		return nil, nil, err
	}

	// Find header
	var hdr *model.Header
	if common.IsLatestBlock(blockNr) {
		hdr, err = srv.FindLatestBlock()
	} else {
		hdr, err = srv.FindBlockByNumber(blockNr)
	}
	if err != nil {
		logger.Error("Failed to find header for block", "err", err)
		return nil, nil, err
	}
	blockNumber := big.NewInt(hdr.Number)

	// Find contract account
	account, err := srv.FindAccount(contractAddress, hdr.Number)
	if err != nil {
		logger.Error("Failed to find contract", "err", err)
		return nil, nil, err
	}

	// Get balance from contract
	db := &contractDB{
		blockNumber:  blockNumber.Int64(),
		code:         erc20,
		account:      account,
		accountStore: srv.accountStore,
	}
	balance, err := BalanceOf(db, contractAddress, address)
	if err != nil {
		logger.Error("Failed to get balance", "err", err)
		return nil, nil, err
	}
	if db.err != nil {
		logger.Error("Failed to get balance due to state db error", "err", db.err)
		return nil, nil, db.err
	}

	// Consider decimals
	result := decimal.NewFromBigInt(balance, -int32(erc20.Decimals))
	return &result, blockNumber, nil
}
