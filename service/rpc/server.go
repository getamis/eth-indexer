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
	"google.golang.org/grpc/reflection"
)

const datetimeFormat string = "2006-01-02 15:04:05.000"

type server struct {
	bhStore bhStore.Store
	txStore txStore.Store
	trStore trStore.Store
	logger  log.Logger
}

func New(db *gorm.DB) *server {
	logger := log.New("ws", "grpc")
	return &server{
		bhStore: bhStore.NewWithDB(db),
		txStore: txStore.NewWithDB(db),
		trStore: trStore.NewWithDB(db),
		logger:  logger,
	}
}

func (s *server) Bind(server *grpc.Server) {
	// register block service
	var bs pb.BlockServiceServer = s
	pb.RegisterBlockServiceServer(server, bs)

	// register transaction service
	var ts pb.TransactionServiceServer = s
	pb.RegisterTransactionServiceServer(server, ts)

	// Register reflection service on gRPC server
	reflection.Register(server)
}

func (s *server) Shutdown() {
	log.Info("Transaction gRPC API shutdown successfully")
}

// Implement grpc functions

func (s *server) GetBlockByHash(ctx context.Context, req *pb.BlockQueryRequest) (*pb.BlockQueryResponse, error) {
	headers, err := s.bhStore.Find(&pb.BlockHeader{
		Hash: req.Hash,
	})
	if err != nil {
		return nil, err
	}
	// get the only block header of results
	header := headers[0]

	// get transactions
	transactions, err := s.txStore.Find(&pb.Transaction{
		BlockHash: req.Hash,
	})
	if err != nil {
		return nil, err
	}
	var tqrs []*pb.TransactionQueryResponse
	for _, transaction := range transactions {
		tqrs = append(tqrs, &pb.TransactionQueryResponse{
			Hash:     transaction.Hash,
			From:     transaction.From,
			To:       transaction.To,
			Nonce:    transaction.Nonce,
			GasPrice: transaction.GasPrice,
			GasLimit: transaction.GasLimit,
			Amount:   transaction.Amount,
			Payload:  transaction.Payload,
		})
	}

	return &pb.BlockQueryResponse{
		Hash:         header.Hash,
		Number:       header.Number,
		Nonce:        header.Nonce,
		Transactions: tqrs,
	}, nil
}

func (s *server) GetTransactionByHash(ctx context.Context, req *pb.TransactionQueryRequest) (*pb.TransactionQueryResponse, error) {
	transactions, err := s.txStore.Find(&pb.Transaction{
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
