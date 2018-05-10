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

package model

import "github.com/ethereum/go-ethereum/common/hexutil"

// Header represents the header of a block
type Header struct {
	Hash        []byte
	ParentHash  []byte
	UncleHash   []byte
	Coinbase    []byte
	Root        []byte
	TxHash      []byte
	ReceiptHash []byte
	Difficulty  int64
	Number      int64
	GasLimit    int64
	GasUsed     int64
	Time        int64
	ExtraData   []byte
	MixDigest   []byte
	Nonce       []byte
	// golang database/sql driver doesn't support uint64, so store the nonce by bytes in db
	// for block header. (only block's nonce may go over int64 range)
	// https://github.com/golang/go/issues/6113
	// https://github.com/golang/go/issues/9373
}

// Transaction represents a transaction
type Transaction struct {
	Hash        []byte
	BlockHash   []byte
	From        []byte
	To          []byte
	Nonce       int64
	GasPrice    string
	GasLimit    int64
	Amount      string
	Payload     []byte
	BlockNumber int64
}

// Receipt represents a transaction receipt
type Receipt struct {
	Root              []byte
	Status            uint
	CumulativeGasUsed int64
	Bloom             []byte
	TxHash            []byte
	ContractAddress   []byte
	GasUsed           int64
	BlockNumber       int64
}

// StateBlock represents the state is at the given block
type TotalDifficulty struct {
	Block int64
	Hash  []byte
	Td    string
}

// Account represents the state of externally owned accounts in Ethereum at given block
type Account struct {
	BlockNumber int64
	Address     []byte
	Balance     string
	Nonce       int64
}

// ERC20 represents the ERC20 contract
type ERC20 struct {
	BlockNumber int64
	Address     []byte
	Code        []byte
	TotalSupply string
	Decimals    int
	Name        string
}

// ERC20Storage represents the contract storage
type ERC20Storage struct {
	Address     []byte `gorm:"-"`
	BlockNumber int64  `gorm:"index;unique_index:idx_block_number_key_hash"`
	Key         []byte `gorm:"column:key_hash;size:32;unique_index:idx_block_number_key_hash"`
	Value       []byte `gorm:"size:32"`
}

// TableName retruns the table name of this erc20 contract
func (s ERC20Storage) TableName() string {
	return ERC20ContractTableName(s.Address)
}

// ERC20ContractTableName returns its contract table
func ERC20ContractTableName(address []byte) string {
	return "erc20_" + hexutil.Encode(address)
}
