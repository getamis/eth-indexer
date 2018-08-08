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

package common

import (
	"encoding/binary"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/getamis/eth-indexer/model"
	"github.com/getamis/sirius/log"
)

const (
	emptyEventName = "NA"
)

// IsLatestBlock returns true if blockNumber < 0 and false otherwise.
func IsLatestBlock(blockNumber int64) bool {
	return blockNumber < 0
}

// Hex returns a hash string and lower-case string without '0x'
func Hex(str string) string {
	return strings.ToLower(strings.TrimPrefix(str, "0x"))
}

// HashHex returns a hash hex and lower-case string without '0x'
func HashHex(hash common.Hash) string {
	return Hex(hash.Hex())
}

// AddressHex returns an address hex and lower-case string without '0x'
func AddressHex(address common.Address) string {
	return Hex(address.Hex())
}

// BytesToHex returns a hex representation (lower-case string without '0x') of a byte array
func BytesToHex(data []byte) string {
	return Hex(hexutil.Encode(data))
}

// BytesTo0xHex returns a hex representation (with '0x') of a byte array
func BytesTo0xHex(data []byte) string {
	return strings.ToLower(hexutil.Encode(data))
}

// HexToBytes returns byte array of a hex string (with or without '0x')
func HexToBytes(hex string) []byte {
	return common.FromHex(hex)
}

// StringToHex returns a hex representation (lower-case string without '0x') of a string
func StringToHex(data string) string {
	return BytesToHex([]byte(data))
}

func ParseTd(ltd *model.TotalDifficulty) (*big.Int, error) {
	td, ok := new(big.Int).SetString(ltd.Td, 10)
	if !ok || td.Cmp(common.Big0) <= 0 {
		return nil, ErrInvalidTD
	}
	return td, nil
}

// TotalDifficulty creates a db struct for an ethereum block
func TotalDifficulty(b *types.Block, td *big.Int) *model.TotalDifficulty {
	return &model.TotalDifficulty{
		Block: b.Number().Int64(),
		Hash:  b.Hash().Bytes(),
		Td:    td.String(),
	}
}

// Header converts ethereum block to db block
func Header(b *types.Block) *model.Header {
	header := b.Header()
	nonce := make([]byte, 8)
	binary.BigEndian.PutUint64(nonce, header.Nonce.Uint64())

	return &model.Header{
		Hash:        b.Hash().Bytes(),
		ParentHash:  header.ParentHash.Bytes(),
		UncleHash:   header.UncleHash.Bytes(),
		Coinbase:    header.Coinbase.Bytes(),
		Root:        header.Root.Bytes(),
		TxHash:      header.TxHash.Bytes(),
		ReceiptHash: header.ReceiptHash.Bytes(),
		Difficulty:  header.Difficulty.Int64(),
		Number:      header.Number.Int64(),
		GasLimit:    int64(header.GasLimit),
		GasUsed:     int64(header.GasUsed),
		Time:        header.Time.Int64(),
		ExtraData:   header.Extra,
		MixDigest:   header.MixDigest.Bytes(),
		Nonce:       nonce,
	}
}

// Transaction converts ethereum transaction to db transaction
func Transaction(chainTest bool, b *types.Block, tx *types.Transaction) (*model.Transaction, error) {
	var signer types.Signer
	if chainTest {
		signer = types.MakeSigner(params.TestChainConfig, b.Number())
	} else {
		signer = types.MakeSigner(params.MainnetChainConfig, b.Number())
	}

	msg, err := tx.AsMessage(signer)
	if err != nil {
		log.Error("Failed to get transaction message", "err", err)
		return &model.Transaction{}, ErrWrongSigner
	}

	t := &model.Transaction{
		Hash:        tx.Hash().Bytes(),
		BlockHash:   b.Hash().Bytes(),
		From:        msg.From().Bytes(),
		Nonce:       int64(msg.Nonce()),
		GasPrice:    msg.GasPrice().Int64(),
		GasLimit:    int64(msg.Gas()),
		Amount:      msg.Value().String(),
		Payload:     msg.Data(),
		BlockNumber: b.Number().Int64(),
	}
	if msg.To() != nil {
		t.To = msg.To().Bytes()
	}
	return t, nil
}

// Receipt converts ethereum transaction receipt to db transaction receipt
func Receipt(b *types.Block, receipt *types.Receipt) (*model.Receipt, error) {
	// Construct receipt model
	r := &model.Receipt{
		Root:              receipt.PostState,
		Status:            uint(receipt.Status),
		CumulativeGasUsed: int64(receipt.CumulativeGasUsed),
		Bloom:             receipt.Bloom.Bytes(),
		TxHash:            receipt.TxHash.Bytes(),
		GasUsed:           int64(receipt.GasUsed),
		BlockNumber:       b.Number().Int64(),
	}
	if receipt.ContractAddress != (common.Address{}) {
		r.ContractAddress = receipt.ContractAddress.Bytes()
	}

	// Construct receipt log model
	var logs []*model.Log
	for _, l := range receipt.Logs {
		// The length of topics should be equal or smaller than 4.
		// 1 event name and at most 3 indexed parameters
		if len(l.Topics) > 4 {
			log.Error("Invalid topic length", "hash", receipt.TxHash.Hex(), "len", len(l.Topics))
			return nil, ErrInvalidReceiptLog
		}

		eventName := []byte(emptyEventName)
		if len(l.Topics) > 0 {
			eventName = l.Topics[0].Bytes()
		}
		log := &model.Log{
			TxHash:          r.TxHash,
			BlockNumber:     r.BlockNumber,
			ContractAddress: l.Address.Bytes(),
			EventName:       eventName,
			Data:            l.Data,
		}
		for i := 1; i < len(l.Topics); i++ {
			switch i {
			case 1:
				log.Topic1 = l.Topics[i].Bytes()
			case 2:
				log.Topic2 = l.Topics[i].Bytes()
			case 3:
				log.Topic3 = l.Topics[i].Bytes()
			}
		}
		logs = append(logs, log)
	}
	r.Logs = logs
	return r, nil
}

// EthTransferEvent converts eth transfer log to eth tranfer event
func EthTransferEvent(b *types.Block, log *types.TransferLog) *model.Transfer {
	return &model.Transfer{
		Address:     model.ETHBytes,
		BlockNumber: b.Number().Int64(),
		TxHash:      log.TxHash.Bytes(),
		From:        log.From.Bytes(),
		To:          log.To.Bytes(),
		Value:       log.Value.String(),
	}
}

// Some weird constants to avoid constant memory allocs for them.
var (
	big8  = big.NewInt(8)
	big32 = big.NewInt(32)
)

// AccumulateRewards credits the coinbase of the given block with the mining
// reward. The total reward consists of the static block reward and rewards for
// included uncles. The coinbase of each uncle block is also rewarded.
//
// **COPIED FROM**: github.com/ethereum/go-ethereum/consensus/ethash/consensus.go#accumulateRewards()
func AccumulateRewards(header *types.Header, uncles []*types.Header) (minerBaseReward, uncleInclusionReward *big.Int, uncleCoinbase []common.Address, uncleReward []*big.Int, uncleHash []common.Hash) {
	// Select the correct block reward based on chain progression
	minerBaseReward = ethash.FrontierBlockReward
	if params.MainnetChainConfig.ByzantiumBlock.Cmp(header.Number) <= 0 {
		minerBaseReward = ethash.ByzantiumBlockReward
	}

	// Accumulate the rewards for the miner and any included uncles
	r := new(big.Int)
	uncleInclusionReward = new(big.Int)
	uncleReward = make([]*big.Int, len(uncles))
	uncleHash = make([]common.Hash, len(uncles))
	uncleCoinbase = make([]common.Address, len(uncles))
	for i, uncle := range uncles {
		r.Add(uncle.Number, big8)
		r.Sub(r, header.Number)
		r.Mul(r, minerBaseReward)
		r.Div(r, big8)
		// store uncle reward
		uncleReward[i] = new(big.Int).Set(r)

		// store uncle inclusion reward
		r.Div(minerBaseReward, big32)
		uncleInclusionReward.Add(uncleInclusionReward, r)

		// store uncle information
		uncleCoinbase[i] = uncle.Coinbase
		uncleHash[i] = uncle.Hash()
	}
	return
}
