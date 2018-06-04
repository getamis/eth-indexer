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
package erc20

import (
	"context"

	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/getamis/sirius/log"
	"github.com/getamis/eth-indexer/service"
	"github.com/getamis/eth-indexer/service/pb"
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
