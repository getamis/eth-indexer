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
	"encoding/binary"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/getamis/sirius/log"
	"github.com/maichain/eth-indexer/client"
	"github.com/maichain/eth-indexer/common"
	"github.com/maichain/eth-indexer/contracts"
	. "github.com/maichain/eth-indexer/service"
	"github.com/maichain/eth-indexer/service/pb"
	"github.com/maichain/mapi/api"
	"github.com/shopspring/decimal"
	"google.golang.org/grpc"
)

type relayServer struct {
	client      client.EthClient
	erc20ABI    abi.ABI
	logger      log.Logger
	middlewares []middleware
}

func NewRelay(client client.EthClient) *relayServer {
	logger := log.New("ws", "relay")
	parsed, _ := abi.JSON(strings.NewReader(contracts.ERC20TokenABI))
	s := &relayServer{
		client:   client,
		erc20ABI: parsed,
		logger:   logger,
	}

	s.middlewares = append(s.middlewares,
		newValidatingMiddleware(logger, s),
	)

	return s
}

func (s *relayServer) Bind(server *grpc.Server) {
	var srv Server = s
	for _, mw := range s.middlewares {
		srv = mw(srv)
	}
	pb.RegisterBlockServiceServer(server, srv)
	pb.RegisterTransactionServiceServer(server, srv)
	pb.RegisterAccountServiceServer(server, srv)
}

func (s *relayServer) Shutdown() {
	log.Info("Relay gRPC API shutdown successfully")
}

// Implement grpc functions
func (s *relayServer) GetBlockByHash(ctx context.Context, req *pb.BlockHashQueryRequest) (*pb.BlockQueryResponse, error) {
	logger := s.logger.New("trackingId", api.GetTrackingIDFromContext(ctx), "hash", req.Hash)
	block, err := s.client.BlockByHash(ctx, ethCommon.HexToHash(req.Hash))
	if err != nil {
		logger.Error("Failed to get block from ethereum", "err", err)
		if err == ethereum.NotFound {
			return nil, ErrBlockNotFound
		}
		return nil, ErrInternal
	}

	response, err := buildBlockQueryResponse(block)
	if err != nil {
		logger.Error("Failed to parse block", "err", err)
		return nil, ErrInternal
	}
	return response, nil
}

func (s *relayServer) GetBlockByNumber(ctx context.Context, req *pb.BlockNumberQueryRequest) (*pb.BlockQueryResponse, error) {
	logger := s.logger.New("trackingId", api.GetTrackingIDFromContext(ctx), "number", req.Number)
	block, err := s.client.BlockByNumber(ctx, new(big.Int).SetInt64(req.Number))
	if err != nil {
		logger.Error("Failed to get block from ethereum", "err", err)
		if err == ethereum.NotFound {
			return nil, ErrBlockNotFound
		}
		return nil, ErrInternal
	}

	response, err := buildBlockQueryResponse(block)
	if err != nil {
		logger.Error("Failed to parse block", "err", err)
		return nil, ErrInternal
	}
	return response, nil
}

func (s *relayServer) GetTransactionByHash(ctx context.Context, req *pb.TransactionQueryRequest) (*pb.TransactionQueryResponse, error) {
	logger := s.logger.New("trackingId", api.GetTrackingIDFromContext(ctx), "hash", req.Hash)
	tx, _, err := s.client.TransactionByHash(ctx, ethCommon.HexToHash(req.Hash))
	if err != nil {
		logger.Error("Failed to get transaction from ethereum", "err", err)
		if err == ethereum.NotFound {
			return nil, ErrTransactionNotFound
		}
		return nil, ErrInternal
	}

	to := ""
	if tx.To() != nil {
		to = common.AddressHex(*tx.To())
	}
	return &pb.TransactionQueryResponse{Tx: &pb.Transaction{
		Hash: tx.Hash().Hex(),
		// TODO: Need block number to get signer
		// From:     common.BytesTo0xHex(transaction.From),
		To:       to,
		Nonce:    int64(tx.Nonce()),
		GasPrice: tx.GasPrice().String(),
		GasLimit: int64(tx.Gas()),
		Amount:   tx.Value().String(),
		Payload:  tx.Data(),
	}}, nil
}

func (s *relayServer) GetBalance(ctx context.Context, req *pb.GetBalanceRequest) (*pb.GetBalanceResponse, error) {
	logger := s.logger.New("trackingId", api.GetTrackingIDFromContext(ctx), "addr", req.Address, "number", req.BlockNumber, "token", req.Token)
	address := ethCommon.HexToAddress(req.Address)
	var blockNumber *big.Int
	if req.BlockNumber < 0 {
		// Get latest block
		block, err := s.client.BlockByNumber(ctx, nil)
		if err != nil {
			logger.Error("Failed to get latest block", "err", err)
			return nil, ErrInternal
		}
		blockNumber = block.Number()
	} else {
		blockNumber = new(big.Int).SetInt64(req.BlockNumber)
	}

	if req.Token == ethToken {
		// Get Ether
		balance, err := s.client.BalanceAt(ctx, address, blockNumber)
		if err != nil {
			logger.Error("Failed to get balance from ethereum", "err", err)
			return nil, ErrInternal
		}

		return &pb.GetBalanceResponse{
			Amount:      balance.String(),
			BlockNumber: blockNumber.Int64(),
		}, nil
	}
	// Get ERC20 token
	balance, err := s.balanceOf(ctx, blockNumber, ethCommon.HexToAddress(req.Token), address)
	if err != nil {
		logger.Error("Failed to get balance from ethereum", "err", err)
		return nil, ErrInternal
	}

	d, err := s.decimals(ctx, blockNumber, ethCommon.HexToAddress(req.Token))
	if err != nil {
		logger.Error("Failed to get decimals from ethereum", "err", err)
		return nil, ErrInternal
	}

	result := decimal.NewFromBigInt(balance, int32(-(*d)))
	return &pb.GetBalanceResponse{
		Amount:      result.String(),
		BlockNumber: blockNumber.Int64(),
	}, nil
}

func (s *relayServer) GetOffsetBalance(ctx context.Context, req *pb.GetOffsetBalanceRequest) (*pb.GetBalanceResponse, error) {
	logger := s.logger.New("trackingId", api.GetTrackingIDFromContext(ctx), "addr", req.Address, "offset", req.Offset, "token", req.Token)
	// Get latest block
	block, err := s.client.BlockByNumber(ctx, nil)
	if err != nil {
		logger.Error("Failed to get latest block", "err", err)
		return nil, ErrInternal
	}

	// Get target block
	target := block.Number().Int64() - req.Offset
	if target < 0 {
		logger.Error("Offset is larger than current header number", "number", block.Number())
		return nil, ErrInvalidOffset
	}

	return s.GetBalance(ctx, &pb.GetBalanceRequest{
		Token:       req.Token,
		Address:     req.Address,
		BlockNumber: target,
	})
}

// balanceOf returns the ERC20 balance of the address
func (s *relayServer) balanceOf(ctx context.Context, blockNumber *big.Int, contractAddress ethCommon.Address, address ethCommon.Address) (*big.Int, error) {
	method := "balanceOf"
	input, err := s.erc20ABI.Pack(method, address)
	if err != nil {
		return nil, err
	}

	msg := ethereum.CallMsg{
		To:   &contractAddress,
		Data: input,
	}

	output, err := s.client.CallContract(ctx, msg, blockNumber)
	if err != nil {
		return nil, err
	}

	result := new(*big.Int)
	err = s.erc20ABI.Unpack(result, method, output)
	if err != nil {
		return nil, err
	}
	return *result, err
}

// TODO: need to handle if the ERC20 doesn't define this function.
// It's not a must function in ERC20.
// decimals returns the number of decimals in ERC20.
func (s *relayServer) decimals(ctx context.Context, blockNumber *big.Int, contractAddress ethCommon.Address) (*uint8, error) {
	method := "decimals"
	input, err := s.erc20ABI.Pack(method)
	if err != nil {
		return nil, err
	}

	msg := ethereum.CallMsg{
		To:   &contractAddress,
		Data: input,
	}

	output, err := s.client.CallContract(ctx, msg, blockNumber)
	if err != nil {
		return nil, err
	}

	result := new(uint8)
	err = s.erc20ABI.Unpack(result, method, output)
	if err != nil {
		return nil, err
	}
	return result, err
}

func buildBlockQueryResponse(block *types.Block) (*pb.BlockQueryResponse, error) {
	nonce := make([]byte, 8)
	binary.BigEndian.PutUint64(nonce, block.Nonce())
	response := &pb.BlockQueryResponse{
		Block: &pb.Block{
			Hash:   common.BytesTo0xHex(block.Hash().Bytes()),
			Number: block.Number().Int64(),
			Nonce:  nonce,
		},
	}
	for _, t := range block.Transactions() {
		transaction, err := common.Transaction(block, t)
		if err != nil {
			return nil, err
		}
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
		response.Txs = append(response.Txs, tx)
	}
	return response, nil
}
