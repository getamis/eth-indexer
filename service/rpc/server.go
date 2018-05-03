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

	"github.com/getamis/sirius/log"
	"github.com/maichain/eth-indexer/common"
	"github.com/maichain/eth-indexer/model"
	"github.com/maichain/eth-indexer/service/pb"
	"github.com/maichain/eth-indexer/store"
	"google.golang.org/grpc"
)

type server struct {
	manager store.ServiceManager
	logger  log.Logger
}

func New(manager store.ServiceManager) *server {
	logger := log.New("ws", "grpc")
	return &server{
		manager: manager,
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

	// register balance service
	var bls pb.AccountServiceServer = s
	pb.RegisterAccountServiceServer(server, bls)
}

func (s *server) Shutdown() {
	log.Info("Transaction gRPC API shutdown successfully")
}

// Implement grpc functions
func (s *server) GetBlockByHash(ctx context.Context, req *pb.BlockQueryRequest) (*pb.BlockQueryResponse, error) {
	hashBytes := common.HexToBytes(req.Hash)
	header, err := s.manager.FindBlockByHash(hashBytes)
	if err != nil {
		return nil, err
	}

	response := &pb.BlockQueryResponse{
		Block: &pb.Block{
			Hash:   common.BytesToHex(header.Hash),
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

func (s *server) GetBlockByNumber(ctx context.Context, req *pb.BlockQueryRequest) (*pb.BlockQueryResponse, error) {
	if req.Number < latestBlockNumber {
		log.Error("Invalid block number")
		return nil, ErrInvalidBlockNumber
	}

	var header *model.Header
	var err error
	if common.IsLatestBlock(req.Number) {
		header, err = s.manager.FindLatestBlock()
	} else {
		header, err = s.manager.FindBlockByNumber(req.Number)
	}
	if err != nil {
		return nil, err
	}

	response := &pb.BlockQueryResponse{
		Block: &pb.Block{
			Hash:   common.BytesToHex(header.Hash),
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

func (s *server) buildTransactionsForBlock(blockHash []byte, resp *pb.BlockQueryResponse) (err error) {
	transactions, err := s.manager.FindTransactionsByBlockHash(blockHash)
	if err != nil {
		return err
	}
	for _, transaction := range transactions {
		tx := &pb.Transaction{
			Hash:     common.BytesToHex(transaction.Hash),
			From:     common.BytesToHex(transaction.From),
			Nonce:    transaction.Nonce,
			GasPrice: transaction.GasPrice,
			GasLimit: transaction.GasLimit,
			Amount:   transaction.Amount,
			Payload:  transaction.Payload,
		}
		if transaction.To != nil {
			tx.To = common.BytesToHex(transaction.To)
		}
		resp.Txs = append(resp.Txs, tx)
	}
	return nil
}

func (s *server) GetTransactionByHash(ctx context.Context, req *pb.TransactionQueryRequest) (*pb.TransactionQueryResponse, error) {
	transaction, err := s.manager.FindTransaction(common.HexToBytes(req.Hash))
	if err != nil {
		return nil, err
	}
	return &pb.TransactionQueryResponse{Tx: &pb.Transaction{
		Hash:     common.BytesToHex(transaction.Hash),
		From:     common.BytesToHex(transaction.From),
		To:       common.BytesToHex(transaction.To),
		Nonce:    transaction.Nonce,
		GasPrice: transaction.GasPrice,
		GasLimit: transaction.GasLimit,
		Amount:   transaction.Amount,
		Payload:  transaction.Payload,
	}}, nil
}
