package indexer

import (
	"context"
	"math/big"

	"github.com/getamis/sirius/log"
	"github.com/maichain/eth-indexer/indexer/pb"
	manager "github.com/maichain/eth-indexer/store/store_manager"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
)

var logger = log.New()
var ctx = context.TODO()

type Indexer interface {
	Start(from int64, to int64) error
}

func NewIndexer(client *ethclient.Client, manager manager.StoreManager) Indexer {
	return &indexer{
		client,
		manager,
	}
}

type indexer struct {
	client  *ethclient.Client
	manager manager.StoreManager
}

func (indexer *indexer) Start(from int64, to int64) error {
	start := big.NewInt(from)
	end := big.NewInt(to)
	for i := new(big.Int).Set(start); i.Cmp(end) <= 0; i.Add(i, big.NewInt(1)) {
		block, err := indexer.client.BlockByNumber(ctx, i)
		if err != nil {
			return err
		}
		// logger.Info("Parse block " + block.String())

		// get block header
		blockHeader := indexer.ParseBlockHeader(block)

		// get transactions
		var (
			transactions = []*pb.Transaction{}
			receipts     = []*pb.TransactionReceipt{}
		)
		for _, tx := range block.Transactions() {
			// logger.Info(tx.String())
			transaction, receipt, err := indexer.ParseTransaction(tx, block.Number())
			if err != nil {
				return err
			}
			transactions = append(transactions, transaction)
			receipts = append(receipts, receipt)
		}

		// insert data into db
		indexer.manager.Upsert(blockHeader, transactions, receipts)
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

		// Transaction
		v, r, s := tx.RawSignatureValues()
		to := ""
		if msg.To() != nil {
			to = msg.To().String()
		}

		t := &pb.Transaction{
			Hash:     tx.Hash().String(),
			From:     msg.From().String(),
			To:       to,
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
		contractAddr := ""
		if receipt.ContractAddress.Big().Int64() != 0 {
			contractAddr = receipt.ContractAddress.String()
		}
		tr := &pb.TransactionReceipt{
			Root:              receipt.PostState,
			Status:            uint32(receipt.Status),
			CumulativeGasUsed: receipt.CumulativeGasUsed,
			Bloom:             receipt.Bloom.Bytes(),
			TxHash:            receipt.TxHash.String(),
			ContractAddress:   contractAddr,
			GasUsed:           receipt.GasUsed,
		}
		return t, tr, nil
	}
	return nil, nil, nil
}
