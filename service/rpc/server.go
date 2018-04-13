package rpc

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/getamis/sirius/log"
	"github.com/jinzhu/gorm"
	"github.com/maichain/eth-indexer/service/account"
	"github.com/maichain/eth-indexer/service/pb"
	bhStore "github.com/maichain/eth-indexer/store/block_header"
	txStore "github.com/maichain/eth-indexer/store/transaction"
	trStore "github.com/maichain/eth-indexer/store/transaction_receipt"
	"google.golang.org/grpc"
)

const datetimeFormat string = "2006-01-02 15:04:05.000"

type server struct {
	accountAPI *account.API
	bhStore    bhStore.Store
	txStore    txStore.Store
	trStore    trStore.Store
	logger     log.Logger
}

func New(db *gorm.DB, ethDB ethdb.Database) *server {
	logger := log.New("ws", "grpc")
	return &server{
		accountAPI: account.NewAPI(ethDB),
		bhStore:    bhStore.NewWithDB(db),
		txStore:    txStore.NewWithDB(db),
		trStore:    trStore.NewWithDB(db),
		logger:     logger,
	}
}

func (s *server) Bind(server *grpc.Server) {
	// register block service
	var bs pb.BlockServiceServer = s
	pb.RegisterBlockServiceServer(server, bs)

	// register transaction service
	var ts pb.TransactionServiceServer = s
	pb.RegisterTransactionServiceServer(server, ts)

	// register balance service
	var bls pb.AccountServiceServer = s
	pb.RegisterAccountServiceServer(server, bls)
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

func (s *server) GetBalance(ctx context.Context, req *pb.GetBalanceRequest) (*pb.GetBalanceResponse, error) {
	log.Info("GetBalance 1", "req", req)
	// Get ETH
	if req.Token == "ETH" {
		amount, err := s.accountAPI.GetBalance(ctx, common.HexToAddress(req.Address), rpc.BlockNumber(req.BlockNumber))
		if err != nil {
			log.Error("Failed to get ETH balance", "addr", req.Address, "number", req.BlockNumber, "err", err)
			return nil, err
		}
		return &pb.GetBalanceResponse{
			Amount: amount.Int64(),
		}, nil
	}
	// Get ERC20 token
	amount, err := s.accountAPI.GetERC20Balance(ctx, common.HexToAddress(req.Token), common.HexToAddress(req.Address), rpc.BlockNumber(req.BlockNumber))
	if err != nil {
		log.Error("Failed to get Token balance", "token", req.Token, "addr", req.Address, "number", req.BlockNumber, "err", err)
		return nil, err
	}
	return &pb.GetBalanceResponse{
		Amount: amount.Int64(),
	}, nil
}
