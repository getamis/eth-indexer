package store

import (
	"encoding/binary"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/getamis/sirius/log"
	"github.com/maichain/eth-indexer/service/pb"
)

// HashHex returns a hash hex and lower-case string without '0x'
func HashHex(hash common.Hash) string {
	return strings.ToLower(strings.TrimPrefix(hash.Hex(), "0x"))
}

// AddressHex returns an address hex and lower-case string without '0x'
func AddressHex(address common.Address) string {
	return strings.ToLower(strings.TrimPrefix(address.Hex(), "0x"))
}

// Header converts ethereum block to db block
func Header(b *types.Block) *pb.BlockHeader {
	header := b.Header()
	nonce := make([]byte, 8)
	binary.BigEndian.PutUint64(nonce, header.Nonce.Uint64())

	bh := &pb.BlockHeader{
		Hash:        HashHex(b.Hash()),
		ParentHash:  HashHex(header.ParentHash),
		UncleHash:   HashHex(header.UncleHash),
		Coinbase:    AddressHex(header.Coinbase),
		Root:        HashHex(header.Root),
		TxHash:      HashHex(header.TxHash),
		ReceiptHash: HashHex(header.ReceiptHash),
		Bloom:       header.Bloom.Bytes(),
		Difficulty:  header.Difficulty.Int64(),
		Number:      header.Number.Int64(),
		GasLimit:    header.GasLimit,
		GasUsed:     header.GasUsed,
		Time:        header.Time.Uint64(),
		ExtraData:   header.Extra,
		MixDigest:   HashHex(header.MixDigest),
		Nonce:       nonce,
	}
	return bh
}

// Transaction converts ethereum transaction to db transaction
func Transaction(b *types.Block, tx *types.Transaction) (*pb.Transaction, error) {
	signer := types.MakeSigner(params.MainnetChainConfig, b.Number())
	msg, err := tx.AsMessage(signer)
	if err != nil {
		log.Error("Failed to get transaction message", "err", err)
		return nil, ErrWrongSigner
	}

	// Transaction values
	v, r, s := tx.RawSignatureValues()
	to := ""
	if msg.To() != nil {
		AddressHex(*msg.To())
	}

	// Why it's nonce
	nonce := make([]byte, 8)
	binary.BigEndian.PutUint64(nonce, msg.Nonce())

	t := &pb.Transaction{
		Hash:      HashHex(tx.Hash()),
		BlockHash: HashHex(b.Hash()),
		From:      AddressHex(msg.From()),
		To:        to,
		Nonce:     nonce,
		GasPrice:  msg.GasPrice().Int64(),
		GasLimit:  msg.Gas(),
		Amount:    msg.Value().Int64(),
		Payload:   msg.Data(),
		V:         v.Int64(),
		R:         r.Int64(),
		S:         s.Int64(),
	}
	return t, nil
}

// Receipt converts ethereum transaction receipt to db transaction receipt
func Receipt(receipt *types.Receipt) *pb.TransactionReceipt {
	contractAddr := ""
	if receipt.ContractAddress != (common.Address{}) {
		contractAddr = AddressHex(receipt.ContractAddress)
	}
	tr := &pb.TransactionReceipt{
		Root:              receipt.PostState,
		Status:            uint32(receipt.Status),
		CumulativeGasUsed: receipt.CumulativeGasUsed,
		Bloom:             receipt.Bloom.Bytes(),
		TxHash:            HashHex(receipt.TxHash),
		ContractAddress:   contractAddr,
		GasUsed:           receipt.GasUsed,
	}
	return tr
}
