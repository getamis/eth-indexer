package indexer

import (
	"context"
	"encoding/binary"
	"math/big"
	"strconv"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/getamis/sirius/log"
	"github.com/maichain/eth-indexer/indexer/pb"
	manager "github.com/maichain/eth-indexer/store/store_manager"

	common "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

var logger = log.New()
var ctx = context.TODO()

//go:generate mockery -name EthClient

type Indexer interface {
	Start(from int64, to int64) error
	Listen(context.Context, chan *types.Header) error
}

type EthClient interface {
	BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error)
	TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error)
	SubscribeNewHead(ctx context.Context, ch chan<- *types.Header) (ethereum.Subscription, error)
}

func NewIndexer(client EthClient, manager manager.StoreManager) Indexer {
	return &indexer{
		client,
		manager,
	}
}

type indexer struct {
	client  EthClient
	manager manager.StoreManager
}

func (indexer *indexer) Listen(ctx context.Context, ch chan *types.Header) error {
	childCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	_, err := indexer.client.SubscribeNewHead(childCtx, ch)
	if err != nil {
		logger.Info("Failed to subscribe event for new header from ethereum", "err", err)
	}

	logger.Info("Listening new header from ethereum")
	for {
		select {
		case head := <-ch:
			var fromBlock int64
			logger.Info("Got new header", "header", head)
			recentBlock, err := indexer.client.BlockByNumber(ctx, head.Number)
			if err != nil {
				logger.Error("Failed to get block by number", "err", err)
				return err
			}

			recent := indexer.ParseBlockHeader(recentBlock)
			header, err := indexer.manager.GetLatestHeader()

			if err != nil {
				logger.Error("Failed to query header table in database", "err", err)
				return err
			}

			if header != nil {
				fromBlock = header.Number
			}

			toBlock := recent.Number

			logger.Info("Begin indexing", "recent", toBlock, "current", fromBlock)
			indexer.Start(fromBlock, toBlock)
		case <-ctx.Done():
			return nil
		}

	}
}

func (indexer *indexer) Start(from int64, to int64) error {
	for i := from; i <= to; i++ {
		num := big.NewInt(i)
		block, err := indexer.client.BlockByNumber(ctx, num)
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
		logger.Info(strconv.FormatInt(blockHeader.Number, 10))
		err1 := indexer.manager.Upsert(blockHeader, transactions, receipts)
		if err1 != nil {
			logger.Error(err1.Error())
		}
	}
	return nil
}

func (indexer *indexer) ParseBlockHeader(b *types.Block) *pb.BlockHeader {
	header := b.Header()
	nonce := make([]byte, 8)
	binary.BigEndian.PutUint64(nonce, header.Nonce.Uint64())

	bh := &pb.BlockHeader{
		Hash:        b.Hash().String(),
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
		Nonce:       nonce,
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
		nonce := make([]byte, 8)
		binary.BigEndian.PutUint64(nonce, msg.Nonce())

		t := &pb.Transaction{
			Hash:     tx.Hash().String(),
			From:     msg.From().String(),
			To:       to,
			Nonce:    nonce,
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
