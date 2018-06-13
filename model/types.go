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

package model

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
)

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

// TableName returns the table name of this model
func (h Header) TableName() string {
	return "block_headers"
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

// TableName returns the table name of this model
func (t Transaction) TableName() string {
	return "transactions"
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
	Logs              []*Log
}

// TableName returns the table name of this model
func (r Receipt) TableName() string {
	return "transaction_receipts"
}

// Log represents a receipt log
type Log struct {
	TxHash          []byte
	BlockNumber     int64
	ContractAddress []byte
	// The sha3 of the event method
	EventName []byte
	// Indexed parameters of event. At most 3 topics.
	Topic1 []byte
	Topic2 []byte
	Topic3 []byte
	Data   []byte
}

// TableName returns the table name of this model
func (l Log) TableName() string {
	return "receipt_logs"
}

// TotalDifficulty represents total difficulty for this block
type TotalDifficulty struct {
	Block int64
	Hash  []byte
	Td    string
}

// TableName returns the table name of this model
func (t TotalDifficulty) TableName() string {
	return "total_difficulty"
}

// Account represents the state of externally owned accounts in Ethereum at given block
type Account struct {
	BlockNumber int64
	Address     []byte
	Balance     string
	Nonce       int64
}

// TableName returns the table name of this model
func (a Account) TableName() string {
	return "accounts"
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

// TableName returns the table name of this model
func (e ERC20) TableName() string {
	return "erc20"
}

// ERC20Storage represents the contract storage
type ERC20Storage struct {
	Address     []byte `gorm:"-"`
	BlockNumber int64  `gorm:"size:8;index;unique_index:idx_block_number_key_hash"`
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

// ERC20Transfer represents the transfer event in erc20
type ERC20Transfer struct {
	Address     []byte `gorm:"-"`
	BlockNumber int64  `gorm:"size:8;index"`
	TxHash      []byte `gorm:"size:32;index"`
	From        []byte `gorm:"size:20;index"`
	To          []byte `gorm:"size:20;index"`
	Value       string `gorm:"size:32"`
}

// TableName retruns the table name of this erc20 contract
func (s ERC20Transfer) TableName() string {
	return ERC20TransferTableName(s.Address)
}

// ERC20TransferTableName returns its contract table
func ERC20TransferTableName(address []byte) string {
	return "erc20_transfer_" + hexutil.Encode(address)
}

// ETHTransfer represents the transfer event in ether
type ETHTransfer struct {
	BlockNumber int64  `gorm:"size:8;index"`
	TxHash      []byte `gorm:"size:32;index"`
	From        []byte `gorm:"size:20;index"`
	To          []byte `gorm:"size:20;index"`
	Value       string `gorm:"size:32"`
}

// TableName retruns the table name of this erc20 contract
func (e ETHTransfer) TableName() string {
	return "eth_transfer"
}
