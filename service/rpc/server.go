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
	"github.com/jinzhu/gorm"
	"github.com/maichain/eth-indexer/common"
	"github.com/maichain/eth-indexer/service/account"
	"github.com/maichain/eth-indexer/service/pb"
	bhStore "github.com/maichain/eth-indexer/store/block_header"
	txStore "github.com/maichain/eth-indexer/store/transaction"
	trStore "github.com/maichain/eth-indexer/store/transaction_receipt"
	"google.golang.org/grpc"
)

var (
	EmptyBlockResponse = pb.BlockQueryResponse{}
	EmptyTxResponse    = pb.TransactionQueryResponse{}
)

type server struct {
	accountAPI account.API
	bhStore    bhStore.Store
	txStore    txStore.Store
	trStore    trStore.Store
	logger     log.Logger
}

func New(db *gorm.DB) *server {
	logger := log.New("ws", "grpc")
	return &server{
		accountAPI: account.NewAPIWithWithDB(db),
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
	header, err := s.bhStore.FindBlockByHash(common.HexToBytes(req.Hash))
	if common.NotFoundError(err) {
		return &EmptyBlockResponse, nil
	}
	if err != nil {
		return nil, err
	}

	response := &pb.BlockQueryResponse{
		Hash:   common.BytesToHex(header.Hash),
		Number: header.Number,
		Nonce:  header.Nonce}

	// get transactions
	transactions, err := s.txStore.FindTransactionsByBlockHash(common.HexToBytes(req.Hash))
	if err != nil {
		return response, err
	}
	//var tqrs []*pb.TransactionQueryResponse
	for _, transaction := range transactions {
		txResponse := &pb.TransactionQueryResponse{
			Hash:     common.BytesToHex(transaction.Hash),
			From:     common.BytesToHex(transaction.From),
			Nonce:    transaction.Nonce,
			GasPrice: transaction.GasPrice,
			GasLimit: transaction.GasLimit,
			Amount:   transaction.Amount,
			Payload:  transaction.Payload,
		}
		if transaction.To != nil {
			txResponse.To = common.BytesToHex(transaction.To)
		}
		response.Transactions = append(response.Transactions, txResponse)
		//tqrs = append(tqrs, txResponse)
	}
	return response, nil
}

func (s *server) GetTransactionByHash(ctx context.Context, req *pb.TransactionQueryRequest) (*pb.TransactionQueryResponse, error) {
	transaction, err := s.txStore.FindTransaction(common.HexToBytes(req.Hash))
	if common.NotFoundError(err) {
		return &EmptyTxResponse, nil
	}
	if err != nil {
		return nil, err
	}

	return &pb.TransactionQueryResponse{
		Hash:     common.BytesToHex(transaction.Hash),
		From:     common.BytesToHex(transaction.From),
		To:       common.BytesToHex(transaction.To),
		Nonce:    transaction.Nonce,
		GasPrice: transaction.GasPrice,
		GasLimit: transaction.GasLimit,
		Amount:   transaction.Amount,
		Payload:  transaction.Payload,
	}, nil
}
