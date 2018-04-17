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

package account

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type API interface {
	// GetBalance returns the amount of wei for the given address in the state of the
	// given block number. If blockNr < 0, the given block is the latest block.
	// Noted that the return block number may be different from the input one because
	// we don't have state in the input one.
	GetBalance(ctx context.Context, address common.Address, blockNr int64) (balance *big.Int, blockNumber *big.Int, err error)

	// GetERC20Balance returns the amount of ERC20 token for the given address in the state of the
	// given block number. If blockNr < 0, the given block is the latest block.
	// Noted that the return block number may be different from the input one because
	// we don't have state in the input one.
	GetERC20Balance(ctx context.Context, contractAddress, address common.Address, blockNr int64) (balance *big.Int, blockNumber *big.Int, err error)
}

func isLatestBlock(num int64) bool {
	return num < 0
}
