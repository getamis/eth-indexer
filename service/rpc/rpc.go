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

type grpcAPI struct {
	bhStore bhStore.Store
	txStore txStore.Store
	trStore trStore.Store
	logger  log.Logger
}

func New(db *gorm.DB) *grpcAPI {
	logger := log.New("ws", "grpc")
	return &grpcAPI{
		bhStore: bhStore.NewWithDB(db),
		txStore: txStore.NewWithDB(db),
		trStore: trStore.NewWithDB(db),
		logger:  logger,
	}
}

func (srv *grpcAPI) Bind(server *grpc.Server) {
	// register block service
	var bs pb.BlockServiceServer = srv
	pb.RegisterBlockServiceServer(server, bs)

	// register transaction service
	var ts pb.TransactionServiceServer = srv
	pb.RegisterTransactionServiceServer(server, ts)
}

func (srv *grpcAPI) Shutdown() {
	log.Info("Transaction gRPC API shutdown successfully")
}

// Implement grpc functions

func (srv *grpcAPI) GetBlockByHash(ctx context.Context, req *pb.BlockQueryRequest) (*pb.BlockQueryResponse, error) {
	return &pb.BlockQueryResponse{}, nil
}

func (srv *grpcAPI) GetTransactionByHash(ctx context.Context, req *pb.TransactionQueryRequest) (*pb.TransactionQueryResponse, error) {
	return &pb.TransactionQueryResponse{}, nil
}
