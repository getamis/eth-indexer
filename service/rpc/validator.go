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

	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/getamis/sirius/log"
	. "github.com/getamis/eth-indexer/service"
	"github.com/getamis/eth-indexer/service/pb"
)

const (
	latestBlockNumber = -1
)

func newValidatingMiddleware(logger log.Logger, server Server) middleware {
	return func(server Server) Server {
		return &validatingMiddleware{
			logger: logger,
			next:   server,
		}
	}
}

type validatingMiddleware struct {
	logger log.Logger
	next   Server
}

// Implement grpc functions
func (mw *validatingMiddleware) GetBlockByHash(ctx context.Context, req *pb.BlockHashQueryRequest) (*pb.BlockQueryResponse, error) {
	if !isHexHash(req.Hash) {
		mw.logger.Error("Invalid hex address", "hash", req.Hash)
		return nil, ErrInvalidHash
	}

	return mw.next.GetBlockByHash(ctx, req)
}

func (mw *validatingMiddleware) GetBlockByNumber(ctx context.Context, req *pb.BlockNumberQueryRequest) (*pb.BlockQueryResponse, error) {
	if req.Number < latestBlockNumber {
		log.Error("Invalid block number", "number", req.Number)
		return nil, ErrInvalidBlockNumber
	}

	return mw.next.GetBlockByNumber(ctx, req)
}

func (mw *validatingMiddleware) GetTransactionByHash(ctx context.Context, req *pb.TransactionQueryRequest) (*pb.TransactionQueryResponse, error) {
	if !isHexHash(req.Hash) {
		mw.logger.Error("Invalid hex address", "hash", req.Hash)
		return nil, ErrInvalidHash
	}
	return mw.next.GetTransactionByHash(ctx, req)
}

func (mw *validatingMiddleware) GetBalance(ctx context.Context, req *pb.GetBalanceRequest) (*pb.GetBalanceResponse, error) {
	if req.BlockNumber < latestBlockNumber {
		log.Error("Invalid block number", "number", req.BlockNumber)
		return nil, ErrInvalidBlockNumber
	}
	if !ethCommon.IsHexAddress(req.Address) {
		log.Error("Invalid address", "address", req.Address)
		return nil, ErrInvalidAddress
	}
	if req.Token != ethToken && !ethCommon.IsHexAddress(req.Token) {
		log.Error("Invalid token", "token", req.Token)
		return nil, ErrInvalidToken
	}
	return mw.next.GetBalance(ctx, req)
}

func (mw *validatingMiddleware) GetOffsetBalance(ctx context.Context, req *pb.GetOffsetBalanceRequest) (*pb.GetBalanceResponse, error) {
	if req.Offset < 0 {
		log.Error("Invalid offset")
		return nil, ErrInvalidOffset
	}
	if !ethCommon.IsHexAddress(req.Address) {
		log.Error("Invalid address", "address", req.Address)
		return nil, ErrInvalidAddress
	}
	if req.Token != ethToken && !ethCommon.IsHexAddress(req.Token) {
		log.Error("Invalid token", "token", req.Token)
		return nil, ErrInvalidToken
	}
	return mw.next.GetOffsetBalance(ctx, req)
}

// isHexHash checks whether it's a valid hash
func isHexHash(s string) bool {
	if hasHexPrefix(s) {
		s = s[2:]
	}

	return len(s) == 2*ethCommon.HashLength && isHex(s)
}

func hasHexPrefix(str string) bool {
	return len(str) >= 2 && str[0] == '0' && (str[1] == 'x' || str[1] == 'X')
}

func isHexCharacter(c byte) bool {
	return ('0' <= c && c <= '9') || ('a' <= c && c <= 'f') || ('A' <= c && c <= 'F')
}

func isHex(str string) bool {
	if len(str)%2 != 0 {
		return false
	}
	for _, c := range []byte(str) {
		if !isHexCharacter(c) {
			return false
		}
	}
	return true
}
