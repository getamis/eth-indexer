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
	"bytes"
	"errors"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

var (
	// ETHAddress represents ether type in address type
	ETHAddress = common.BytesToAddress([]byte("ETH"))
	// ETHBytes represents ether type in bytes array type
	ETHBytes = ETHAddress.Bytes()
	// RewardToMiner represents a constant at from field in transfer event
	RewardToMiner = common.BytesToAddress([]byte("MINER REWARD"))
	// RewardToUncle represents a constant at from field in transfer event
	RewardToUncle = common.BytesToAddress([]byte("UNCLE REWARD"))

	// Maximum number of uncles allowed in a single block
	MaxUncles = 2
	// ErrTooManyUncles is returned if uncles is larger than 2
	ErrTooManyUncles = errors.New("too many uncles")
	// ErrTooManyMiners is returned if miner is larger than 1
	ErrTooManyMiners  = errors.New("too many miners")
	ErrConfusedUncles = errors.New("confused numbers of uncle")
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
	// MinerBaseReward plus UnclesInclusionReward plus TxsFee is MinerReward.
	MinerReward           string
	UnclesInclusionReward string
	TxsFee                string
	// Total of uncles reward. At most 2.
	Uncle1Reward   string
	Uncle1Coinbase []byte
	Uncle1Hash     []byte
	Uncle2Reward   string
	Uncle2Coinbase []byte
	Uncle2Hash     []byte

	// golang database/sql driver doesn't support uint64, so store the nonce by bytes in db
	// for block header. (only block's nonce may go over int64 range)
	// https://github.com/golang/go/issues/6113
	// https://github.com/golang/go/issues/9373

	CreatedAt *time.Time
}

// TableName returns the table name of this model
func (h Header) TableName() string {
	return "block_headers"
}

// AddReward adds reward to header.
// Verify that there are at most 2 uncles
func (h Header) AddReward(txsFee, minerBaseReward, uncleInclusionReward *big.Int, unclesReward []*big.Int, uncleCBs []common.Address, unclesHash []common.Hash) (*Header, error) {
	if len(unclesReward) != len(unclesHash) || len(unclesReward) != len(uncleCBs) {
		return nil, ErrConfusedUncles
	}
	if len(unclesReward) > MaxUncles {
		return nil, ErrTooManyUncles
	}

	urd := []*big.Int{big.NewInt(0), big.NewInt(0)}
	ush := [][]byte{{}, {}}
	ucb := [][]byte{{}, {}}
	// We assume that the length of coinbases, rewards and hashes are the same.
	for i, u := range unclesHash {
		urd[i] = unclesReward[i]
		ush[i] = u.Bytes()
		ucb[i] = uncleCBs[i].Bytes()
	}
	minerReward := new(big.Int).Add(txsFee, minerBaseReward)
	minerReward.Add(minerReward, uncleInclusionReward)

	h.MinerReward = minerReward.String()
	h.UnclesInclusionReward = uncleInclusionReward.String()
	h.TxsFee = txsFee.String()
	h.Uncle1Reward = urd[0].String()
	h.Uncle1Hash = ush[0]
	h.Uncle1Coinbase = ucb[0]
	h.Uncle2Reward = urd[1].String()
	h.Uncle2Hash = ush[1]
	h.Uncle2Coinbase = ucb[1]
	return &h, nil
}

// Transaction represents a transaction
type Transaction struct {
	Hash        []byte
	BlockHash   []byte
	From        []byte
	To          []byte
	Nonce       int64
	GasPrice    int64
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

// Account represents the either ERC20 or ETH balances of externally owned accounts in Ethereum at given block
// The account is considered an eth account and insert to account table if ContractAddress is ETHBytes, or
// considered an erc20 account and insert to erc20_balance_{ContractAddress} table.
type Account struct {
	ContractAddress []byte `gorm:"-"`
	BlockNumber     int64  `gorm:"size:8;index;unique_index:idx_block_number_address"`
	Address         []byte `gorm:"size:20;index;unique_index:idx_block_number_address"`
	Balance         string `gorm:"size:32"`
}

// TableName returns the table name of this model
func (a Account) TableName() string {
	if bytes.Equal(a.ContractAddress, ETHBytes) {
		return "accounts"
	}
	return "erc20_balance_" + hexutil.Encode(a.ContractAddress)
}

// Transfer represents the transfer event in either ether or ERC20 tokens
// The event is considered an eth transfer event and insert to eth_transfer table if Address is ETHBytes, or
// considered an erc20 transfer event and insert to erc20_transfer_{Address} table.
type Transfer struct {
	Address     []byte `gorm:"-"`
	BlockNumber int64  `gorm:"size:8;index"`
	TxHash      []byte `gorm:"size:32;index"`
	From        []byte `gorm:"size:20;index"`
	To          []byte `gorm:"size:20;index"`
	Value       string `gorm:"size:32"`
}

// TableName retruns the table name of this model
func (e Transfer) TableName() string {
	if bytes.Equal(e.Address, ETHBytes) {
		return "eth_transfer"
	}
	return "erc20_transfer_" + hexutil.Encode(e.Address)
}

// IsMinerRewardEvent represents a miner or uncle event.
//
// Note that the event is defined by us. It's not a standard ethereum event.
func (e Transfer) IsMinerRewardEvent() bool {
	return bytes.Equal(e.From, RewardToMiner.Bytes())
}

// IsUncleRewardEvent represents a miner or uncle event.
//
// Note that the event is defined by us. It's not a standard ethereum event.
func (e Transfer) IsUncleRewardEvent() bool {
	return bytes.Equal(e.From, RewardToUncle.Bytes())
}

// TotalBalance represents the total balance of subscription accounts in different group
type TotalBalance struct {
	Token        []byte
	BlockNumber  int64
	Group        int64
	Balance      string
	TxFee        string
	MinerReward  string
	UnclesReward string
}

// TableName retruns the table name of this model
func (s TotalBalance) TableName() string {
	return "total_balances"
}

// ERC20 represents the ERC20 contract
type ERC20 struct {
	BlockNumber int64
	Address     []byte
	TotalSupply string
	Decimals    int
	Name        string
}

// TableName returns the table name of this model
func (e ERC20) TableName() string {
	return "erc20"
}

// Subscription represents the Subscription model
type Subscription struct {
	ID          int64
	BlockNumber int64
	Group       int64
	Address     []byte
	CreatedAt   time.Time `deepequal:"-"`
	UpdatedAt   time.Time `deepequal:"-"`
}

// TableName retruns the table name of this erc20 contract
func (s Subscription) TableName() string {
	return "subscriptions"
}
