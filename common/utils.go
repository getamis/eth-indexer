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
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/getamis/eth-indexer/model"
	"github.com/getamis/sirius/log"
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
func Transaction(b *types.Block, tx *types.Transaction) (*model.Transaction, error) {
	signer := types.MakeSigner(params.MainnetChainConfig, b.Number())
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
		GasPrice:    msg.GasPrice().String(),
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
func Receipt(b *types.Block, receipt *types.Receipt) *model.Receipt {
	r := &model.Receipt{
		Root:              receipt.PostState,
		Status:            receipt.Status,
		CumulativeGasUsed: int64(receipt.CumulativeGasUsed),
		Bloom:             receipt.Bloom.Bytes(),
		TxHash:            receipt.TxHash.Bytes(),
		GasUsed:           int64(receipt.GasUsed),
		BlockNumber:       b.Number().Int64(),
	}
	if receipt.ContractAddress != (common.Address{}) {
		r.ContractAddress = receipt.ContractAddress.Bytes()
	}
	return r
}
