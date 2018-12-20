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

// Ensure the timezone in model is UTC
func init() {
	time.Local = time.UTC
}

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
	ID          int64  `db:"id"`
	Hash        []byte `db:"hash"`
	ParentHash  []byte `db:"parent_hash"`
	UncleHash   []byte `db:"uncle_hash"`
	Coinbase    []byte `db:"coinbase"`
	Root        []byte `db:"root"`
	TxHash      []byte `db:"tx_hash"`
	ReceiptHash []byte `db:"receipt_hash"`
	Difficulty  int64  `db:"difficulty"`
	Number      int64  `db:"number"`
	GasLimit    int64  `db:"gas_limit"`
	GasUsed     int64  `db:"gas_used"`
	Time        int64  `db:"time"`
	ExtraData   []byte `db:"extra_data"`
	MixDigest   []byte `db:"mix_digest"`
	Nonce       []byte `db:"nonce"`
	// MinerBaseReward plus UnclesInclusionReward plus TxsFee is MinerReward.
	MinerReward           string `db:"miner_reward"`
	UnclesInclusionReward string `db:"uncles_inclusion_reward"`
	TxsFee                string `db:"txs_fee"`
	// Total of uncles reward. At most 2.
	Uncle1Reward   string `db:"uncle1_reward"`
	Uncle1Coinbase []byte `db:"uncle1_coinbase"`
	Uncle1Hash     []byte `db:"uncle1_hash"`
	Uncle2Reward   string `db:"uncle2_reward"`
	Uncle2Coinbase []byte `db:"uncle2_coinbase"`
	Uncle2Hash     []byte `db:"uncle2_hash"`

	// golang database/sql driver doesn't support uint64, so store the nonce by bytes in db
	// for block header. (only block's nonce may go over int64 range)
	// https://github.com/golang/go/issues/6113
	// https://github.com/golang/go/issues/9373

	CreatedAt *time.Time `db:"created_at"`
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
	Hash        []byte `db:"hash"`
	BlockHash   []byte `db:"block_hash"`
	From        []byte `db:"from"`
	To          []byte `db:"to"`
	Nonce       int64  `db:"nonce"`
	GasPrice    int64  `db:"gas_price"`
	GasLimit    int64  `db:"gas_limit"`
	Amount      string `db:"amount"`
	Payload     []byte `db:"payload"`
	BlockNumber int64  `db:"block_number"`
}

// Receipt represents a transaction receipt
type Receipt struct {
	Root              []byte `db:"root"`
	Status            uint   `db:"status"`
	CumulativeGasUsed int64  `db:"cumulative_gas_used"`
	Bloom             []byte `db:"bloom"`
	TxHash            []byte `db:"tx_hash"`
	ContractAddress   []byte `db:"contract_address"`
	GasUsed           int64  `db:"gas_used"`
	BlockNumber       int64  `db:"block_number"`
	Logs              []*Log
}

// Log represents a receipt log
type Log struct {
	TxHash          []byte `db:"tx_hash"`
	BlockNumber     int64  `db:"block_number"`
	ContractAddress []byte `db:"contract_address"`
	// The sha3 of the event method
	EventName []byte `db:"event_name"`
	// Indexed parameters of event. At most 3 topics.
	Topic1 []byte `db:"topic1"`
	Topic2 []byte `db:"topic2"`
	Topic3 []byte `db:"topic3"`
	Data   []byte `db:"data"`
}

// TotalDifficulty represents total difficulty for this block
type TotalDifficulty struct {
	Block int64  `db:"block"`
	Hash  []byte `db:"hash"`
	Td    string `db:"td"`
}

// Account represents the either ERC20 or ETH balances of externally owned accounts in Ethereum at given block
// The account is considered an eth account and insert to account table if ContractAddress is ETHBytes, or
// considered an erc20 account and insert to erc20_balance_{ContractAddress} table.
type Account struct {
	ContractAddress []byte
	BlockNumber     int64  `db:"block_number"`
	Address         []byte `db:"address"`
	Balance         string `db:"balance"`
	Group           int64  `db:"group"`
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
	Address     []byte
	BlockNumber int64  `db:"block_number"`
	TxHash      []byte `db:"tx_hash"`
	From        []byte `db:"from"`
	To          []byte `db:"to"`
	Value       string `db:"value"`
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
	Token        []byte `db:"token"`
	BlockNumber  int64  `db:"block_number"`
	Group        int64  `db:"group"`
	Balance      string `db:"balance"`
	TxFee        string `db:"tx_fee"`
	MinerReward  string `db:"miner_reward"`
	UnclesReward string `db:"uncles_reward"`
}

// ERC20 represents the ERC20 contract
type ERC20 struct {
	BlockNumber int64  `db:"block_number"`
	Address     []byte `db:"address"`
	TotalSupply string `db:"total_supply"`
	Decimals    int    `db:"decimals"`
	Name        string `db:"name"`
}

// Subscription represents the Subscription model
type Subscription struct {
	ID          int64     `db:"id"`
	BlockNumber int64     `db:"block_number"`
	Group       int64     `db:"group"`
	Address     []byte    `db:"address"`
	CreatedAt   time.Time `db:"created_at" deepequal:"-"`
	UpdatedAt   time.Time `db:"updated_at" deepequal:"-"`
}

// Reorg represents the Reorg model
type Reorg struct {
	From      int64     `db:"from"`
	FromHash  []byte    `db:"from_hash"`
	To        int64     `db:"to"`
	ToHash    []byte    `db:"to_hash"`
	CreatedAt time.Time `db:"created_at" deepequal:"-"`
}
