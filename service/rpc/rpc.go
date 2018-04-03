package rpc

import (
	"context"

	"github.com/getamis/sirius/log"
	"github.com/jinzhu/gorm"
	"github.com/maichain/eth-indexer/service/pb"
	bhStore "github.com/maichain/eth-indexer/store/block_header"
	txStore "github.com/maichain/eth-indexer/store/transaction"
	trStore "github.com/maichain/eth-indexer/store/transaction_receipt"
	"google.golang.org/grpc"
)

const datetimeFormat string = "2006-01-02 15:04:05.000"

type rpc struct {
	bhStore bhStore.Store
	txStore txStore.Store
	trStore trStore.Store
	logger  log.Logger
}

func New(db *gorm.DB) *rpc {
	logger := log.New("ws", "grpc")
	return &rpc{
		bhStore: bhStore.NewWithDB(db),
		txStore: txStore.NewWithDB(db),
		trStore: trStore.NewWithDB(db),
		logger:  logger,
	}
}

func (srv *rpc) Bind(server *grpc.Server) {
	// register block service
	var bs pb.BlockServiceServer = srv
	pb.RegisterBlockServiceServer(server, bs)

	// register transaction service
	var ts pb.TransactionServiceServer = srv
	pb.RegisterTransactionServiceServer(server, ts)
}

func (srv *rpc) Shutdown() {
	log.Info("Transaction gRPC API shutdown successfully")
}

// Implement grpc functions

func (srv *rpc) GetBlockByHash(ctx context.Context, req *pb.BlockQueryRequest) (*pb.BlockQueryResponse, error) {
	headers, err := srv.bhStore.Find(&pb.BlockHeader{
		Hash: req.Hash,
	})
	if err != nil {
		return nil, err
	}
	// get the only block header of results
	header := headers[0]

	// get transactions
	transactions, err := srv.txStore.Find(&pb.Transaction{
		Hash: req.Hash,
	})
	if err != nil {
		return nil, err
	}

	// get receipts
	receipts, err := srv.trStore.Find(&pb.TransactionReceipt{
		TxHash: req.Hash,
	})
	if err != nil {
		return nil, err
	}

	return &pb.BlockQueryResponse{
		Hash:         header.Hash,
		Number:       header.Number,
		Nonce:        header.Nonce,
		Transactions: transactions,
		Receipts:     receipts,
	}, nil
}

func (srv *rpc) GetTransactionByHash(ctx context.Context, req *pb.TransactionQueryRequest) (*pb.TransactionQueryResponse, error) {
	transactions, err := srv.txStore.Find(&pb.Transaction{
		Hash: req.Hash,
	})
	if err != nil {
		return nil, err
	}
	// get the only transaction of results
	transaction := transactions[0]

	return &pb.TransactionQueryResponse{
		Hash:     transaction.Hash,
		From:     transaction.From,
		To:       transaction.To,
		Nonce:    transaction.Nonce,
		GasPrice: transaction.GasPrice,
		GasLimit: transaction.GasLimit,
		Amount:   transaction.Amount,
		Payload:  transaction.Payload,
	}, nil
}
