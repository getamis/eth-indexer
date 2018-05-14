// Copyright © 2018 AMIS Technologies
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package rpc

import (
	"context"
	"math/big"

	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/getamis/sirius/log"
	"github.com/maichain/eth-indexer/common"
	"github.com/maichain/eth-indexer/model"
	. "github.com/maichain/eth-indexer/service"
	"github.com/maichain/eth-indexer/service/pb"
	"github.com/maichain/eth-indexer/store"
	"github.com/maichain/mapi/api"
	"google.golang.org/grpc"
)

const (
	ethToken = "ETH"
)

type server struct {
	manager     store.ServiceManager
	logger      log.Logger
	middlewares []middleware
}

func New(manager store.ServiceManager) *server {
	logger := log.New("ws", "grpc")
	s := &server{
		manager: manager,
		logger:  logger,
	}
	s.middlewares = append(s.middlewares,
		newValidatingMiddleware(logger, s),
	)
	return s
}

func (s *server) Bind(server *grpc.Server) {
	var srv Server = s
	for _, mw := range s.middlewares {
		srv = mw(srv)
	}
	pb.RegisterBlockServiceServer(server, srv)
	pb.RegisterTransactionServiceServer(server, srv)
	pb.RegisterAccountServiceServer(server, srv)
}

func (s *server) Shutdown() {
	log.Info("Transaction gRPC API shutdown successfully")
}

// Implement grpc functions
func (s *server) GetBlockByHash(ctx context.Context, req *pb.BlockHashQueryRequest) (*pb.BlockQueryResponse, error) {
	hashBytes := common.HexToBytes(req.Hash)
	header, err := s.manager.FindBlockByHash(hashBytes)
	if err != nil {
		return nil, WrapBlockNotFoundError(err)
	}

	response := &pb.BlockQueryResponse{
		Block: &pb.Block{
			Hash:   common.BytesTo0xHex(header.Hash),
			Number: header.Number,
			Nonce:  header.Nonce},
	}

	// get transactions
	err = s.buildTransactionsForBlock(hashBytes, response)
	if err != nil {
		return response, err
	}
	return response, nil
}

func (s *server) GetBlockByNumber(ctx context.Context, req *pb.BlockNumberQueryRequest) (*pb.BlockQueryResponse, error) {
	var header *model.Header
	var err error
	if common.IsLatestBlock(req.Number) {
		header, err = s.manager.FindLatestBlock()
	} else {
		header, err = s.manager.FindBlockByNumber(req.Number)
	}
	if err != nil {
		return nil, WrapBlockNotFoundError(err)
	}

	response := &pb.BlockQueryResponse{
		Block: &pb.Block{
			Hash:   common.BytesTo0xHex(header.Hash),
			Number: header.Number,
			Nonce:  header.Nonce},
	}

	// get transactions
	err = s.buildTransactionsForBlock(header.Hash, response)
	if err != nil {
		return response, err
	}
	return response, nil
}

func (s *server) GetTransactionByHash(ctx context.Context, req *pb.TransactionQueryRequest) (*pb.TransactionQueryResponse, error) {
	transaction, err := s.manager.FindTransaction(common.HexToBytes(req.Hash))
	if err != nil {
		return nil, WrapTransactionNotFoundError(err)
	}
	result := &pb.TransactionQueryResponse{Tx: &pb.Transaction{
		Hash:     common.BytesTo0xHex(transaction.Hash),
		From:     common.BytesTo0xHex(transaction.From),
		Nonce:    transaction.Nonce,
		GasPrice: transaction.GasPrice,
		GasLimit: transaction.GasLimit,
		Amount:   transaction.Amount,
		Payload:  transaction.Payload,
	}}
	if len(transaction.To) > 0 {
		result.Tx.To = common.BytesTo0xHex(transaction.To)
	}
	return result, nil
}

func (s *server) GetBalance(ctx context.Context, req *pb.GetBalanceRequest) (*pb.GetBalanceResponse, error) {
	logger := s.logger.New("trackingId", api.GetTrackingIDFromContext(ctx), "addr", req.Address, "number", req.BlockNumber, "token", req.Token)
	res, err := s.getBalance(ctx, req.BlockNumber, req.Address, req.Token)
	if err != nil {
		logger.Error("Failed to get balance", "err", err)
		return nil, ErrInternal
	}
	return res, nil
}

func (s *server) GetOffsetBalance(ctx context.Context, req *pb.GetOffsetBalanceRequest) (*pb.GetBalanceResponse, error) {
	logger := s.logger.New("trackingId", api.GetTrackingIDFromContext(ctx), "addr", req.Address, "offset", req.Offset, "token", req.Token)
	// Get latest block
	header, err := s.manager.FindLatestBlock()
	if err != nil {
		log.Error("Failed to get latest header", "err", err)
		return nil, ErrInternal
	}

	// Get target block
	target := header.Number - req.Offset
	if target < 0 {
		log.Error("Offset is larger than current header number", "number", header.Number)
		return nil, ErrInvalidOffset
	}

	res, err := s.getBalance(ctx, target, req.Address, req.Token)
	if err != nil {
		logger.Error("Failed to get balance", "err", err)
		return nil, ErrInternal
	}
	return res, nil
}

func (s *server) getBalance(ctx context.Context, blockNr int64, addr string, token string) (*pb.GetBalanceResponse, error) {
	// Get balance
	var err error
	var number *big.Int
	var balance *big.Int
	if token == ethToken {
		// Get Ether
		balance, number, err = s.manager.GetBalance(ctx, ethCommon.HexToAddress(addr), blockNr)
	} else {
		// Get ERC20 token
		balance, number, err = s.manager.GetERC20Balance(ctx, ethCommon.HexToAddress(token), ethCommon.HexToAddress(addr), blockNr)
	}
	if err != nil {
		return nil, err
	}
	return &pb.GetBalanceResponse{
		Amount:      balance.String(),
		BlockNumber: number.Int64(),
	}, nil
}

func (s *server) buildTransactionsForBlock(blockHash []byte, resp *pb.BlockQueryResponse) (err error) {
	transactions, err := s.manager.FindTransactionsByBlockHash(blockHash)
	if err != nil {
		return err
	}
	for _, transaction := range transactions {
		tx := &pb.Transaction{
			Hash:     common.BytesTo0xHex(transaction.Hash),
			From:     common.BytesTo0xHex(transaction.From),
			Nonce:    transaction.Nonce,
			GasPrice: transaction.GasPrice,
			GasLimit: transaction.GasLimit,
			Amount:   transaction.Amount,
			Payload:  transaction.Payload,
		}
		if transaction.To != nil {
			tx.To = common.BytesTo0xHex(transaction.To)
		}
		resp.Txs = append(resp.Txs, tx)
	}
	return nil
}
