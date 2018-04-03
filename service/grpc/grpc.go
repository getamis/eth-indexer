package grpc

import (
	"context"

	"github.com/getamis/sirius/log"
	"github.com/maichain/eth-indexer/service/pb"
	manager "github.com/maichain/eth-indexer/store/store_manager"
	"google.golang.org/grpc"
)

const datetimeFormat string = "2006-01-02 15:04:05.000"

type grpcAPI struct {
	manager manager.StoreManager
	logger  log.Logger
}

func NewIndexer(manager manager.StoreManager) *grpcAPI {
	logger := log.New("ws", "grpc")
	return &grpcAPI{
		manager,
		logger,
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
