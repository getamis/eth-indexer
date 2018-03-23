package indexer

import (
	"context"
	"math/big"

	"github.com/getamis/sirius/log"
	"github.com/maichain/eth-indexer/indexer/pb"
	store "github.com/maichain/eth-indexer/store/block_header"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
)

var logger = log.New()
var ctx = context.TODO()

type Indexer interface {
	Start() error
}

func NewIndexer(client *ethclient.Client, store store.Store) Indexer {
	return &indexer{
		client,
		store,
	}
}

type indexer struct {
	client *ethclient.Client
	store  store.Store
}

func (indexer *indexer) Start() error {
	ctx := context.TODO()

	start := big.NewInt(2000000)
	end := big.NewInt(2000003)
	for i := new(big.Int).Set(start); i.Cmp(end) <= 0; i.Add(i, big.NewInt(1)) {
		block, err := indexer.client.BlockByNumber(ctx, i)
		// logger.Info("Parse block " + block.String())

		if err != nil {
			return err
		}

		for _, tx := range block.Transactions() {
			logger.Info(tx.String())
			indexer.ParseTransaction(tx, block.Number())
		}
	}
	return nil
}

func (indexer *indexer) ParseBlockHeader(b *types.Block) *pb.BlockHeader {
	header := b.Header()
	bh := &pb.BlockHeader{
		ParentHash:  header.ParentHash.String(),
		UncleHash:   header.UncleHash.String(),
		Coinbase:    header.Coinbase.String(),
		Root:        header.Root.String(),
		TxHash:      header.TxHash.String(),
		ReceiptHash: header.ReceiptHash.String(),
		Bloom:       header.Bloom.Bytes(),
		Difficulty:  header.Difficulty.Int64(),
		Number:      header.Number.Int64(),
		GasLimit:    header.GasLimit,
		GasUsed:     header.GasUsed,
		Time:        header.Time.Uint64(),
		ExtraData:   header.Extra,
		MixDigest:   header.MixDigest.String(),
		Nonce:       header.Nonce.Uint64(),
	}
	return bh
}

func (indexer *indexer) ParseTransaction(tx *types.Transaction, blockNumber *big.Int) (*pb.Transaction, *pb.TransactionReceipt, error) {
	receipt, err := indexer.client.TransactionReceipt(ctx, tx.Hash())
	if err != nil {
		return nil, nil, err
	}

	// the transactions in block must include a receipt
	if receipt != nil {
		signer := types.MakeSigner(params.MainnetChainConfig, blockNumber)
		msg, err := tx.AsMessage(signer)
		if err != nil {
			return nil, nil, err
		}

		v, r, s := tx.RawSignatureValues()

		// Transaction
		t := &pb.Transaction{
			Hash:     tx.Hash().String(),
			From:     msg.From().String(),
			To:       msg.To().String(),
			Nonce:    msg.Nonce(),
			GasPrice: msg.GasPrice().Int64(),
			GasLimit: msg.Gas(),
			Amount:   msg.Value().Int64(),
			Payload:  msg.Data(),
			V:        v.Int64(),
			R:        r.Int64(),
			S:        s.Int64(),
		}

		// Receipt
		tr := &pb.TransactionReceipt{
			Root:              receipt.PostState,
			Status:            uint32(receipt.Status),
			CumulativeGasUsed: receipt.CumulativeGasUsed,
			Bloom:             receipt.Bloom.Bytes(),
			TxHash:            receipt.TxHash.String(),
			ContractAddress:   receipt.ContractAddress.String(),
			GasUsed:           receipt.GasUsed,
		}
		return t, tr, nil
	}
	return nil, nil, nil
}
