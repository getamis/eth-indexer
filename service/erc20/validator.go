// Copyright 2018 The eth-indexer Authors
// This file is part of the eth-indexer library.
//
// The eth-indexer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The eth-indexer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the eth-indexer library. If not, see <http://www.gnu.org/licenses/>.

package erc20

import (
	"context"

	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/getamis/eth-indexer/service"
	"github.com/getamis/eth-indexer/service/pb"
	"github.com/getamis/sirius/log"
)

func newValidatingMiddleware(logger log.Logger, server pb.ERC20ServiceServer) middleware {
	return func(server pb.ERC20ServiceServer) pb.ERC20ServiceServer {
		return &validatingMiddleware{
			logger: logger,
			next:   server,
		}
	}
}

type validatingMiddleware struct {
	logger log.Logger
	next   pb.ERC20ServiceServer
}

func (mw *validatingMiddleware) AddERC20(ctx context.Context, req *pb.AddERC20Request) (res *pb.AddERC20Response, err error) {
	if !ethCommon.IsHexAddress(req.Address) {
		log.Error("Invalid address", "address", req.Address)
		return nil, service.ErrInvalidAddress
	}
	if req.BlockNumber < 0 {
		log.Error("Invalid block number", "number", req.BlockNumber)
		return nil, service.ErrInvalidBlockNumber
	}
	return mw.next.AddERC20(ctx, req)
}
