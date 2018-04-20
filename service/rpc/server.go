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
	"github.com/maichain/eth-indexer/model"
	"github.com/maichain/eth-indexer/service/account"
	"github.com/maichain/eth-indexer/service/pb"
	bhStore "github.com/maichain/eth-indexer/store/block_header"
	txStore "github.com/maichain/eth-indexer/store/transaction"
	trStore "github.com/maichain/eth-indexer/store/transaction_receipt"
	"google.golang.org/grpc"
)

const (
	datetimeFormat = "2006-01-02 15:04:05.000"
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
	headers, err := s.bhStore.Find(&model.Header{
		Hash: common.HexToBytes(req.Hash),
	})
	if err != nil {
		return nil, err
	}
	// get the only block header of results
	header := headers[0]

	// get transactions
	transactions, err := s.txStore.Find(&model.Transaction{
		BlockHash: common.HexToBytes(req.Hash),
	})
	if err != nil {
		return nil, err
	}
	var tqrs []*pb.TransactionQueryResponse
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
		tqrs = append(tqrs, txResponse)
	}

	return &pb.BlockQueryResponse{
		Hash:         common.BytesToHex(header.Hash),
		Number:       header.Number,
		Nonce:        header.Nonce,
		Transactions: tqrs,
	}, nil
}

func (s *server) GetTransactionByHash(ctx context.Context, req *pb.TransactionQueryRequest) (*pb.TransactionQueryResponse, error) {
	transactions, err := s.txStore.Find(&model.Transaction{
		Hash: common.HexToBytes(req.Hash),
	})
	if err != nil {
		return nil, err
	}
	// get the only transaction of results
	transaction := transactions[0]

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
