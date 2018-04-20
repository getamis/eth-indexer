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

package common

import (
	"encoding/binary"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/getamis/sirius/log"
	"github.com/maichain/eth-indexer/model"
)

// Hex returns a hash string and lower-case string without '0x'
func Hex(str string) string {
	return strings.ToLower(strings.TrimPrefix(str, "0x"))
}

// HashHex returns a hash hex and lower-case string without '0x'
func HashHex(hash common.Hash) string {
	return Hex(hash.Hex())
}

// BytesToHex returns a hex representation (lower-case string without '0x') of a byte array
func BytesToHex(data []byte) string {
	return Hex(hexutil.Encode(data))
}

// HexToBytes returns byte array of a hex string (with or without '0x')
func HexToBytes(hex string) []byte {
	return common.FromHex(hex)
}

// StringToHex returns a hex representation (lower-case string without '0x') of a string
func StringToHex(data string) string {
	return BytesToHex([]byte(data))
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
		Hash:      tx.Hash().Bytes(),
		BlockHash: b.Hash().Bytes(),
		From:      msg.From().Bytes(),
		Nonce:     int64(msg.Nonce()),
		GasPrice:  msg.GasPrice().String(),
		GasLimit:  int64(msg.Gas()),
		Amount:    msg.Value().String(),
		Payload:   msg.Data(),
	}
	if msg.To() != nil {
		t.To = msg.To().Bytes()
	}
	return t, nil
}

// Receipt converts ethereum transaction receipt to db transaction receipt
func Receipt(receipt *types.Receipt) *model.Receipt {
	r := &model.Receipt{
		Root:              receipt.PostState,
		Status:            receipt.Status,
		CumulativeGasUsed: int64(receipt.CumulativeGasUsed),
		Bloom:             receipt.Bloom.Bytes(),
		TxHash:            receipt.TxHash.Bytes(),
		GasUsed:           int64(receipt.GasUsed),
	}
	if receipt.ContractAddress != (common.Address{}) {
		r.ContractAddress = receipt.ContractAddress.Bytes()
	}
	return r
}
