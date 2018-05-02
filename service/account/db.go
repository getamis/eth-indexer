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
	"errors"
	"math/big"

	gethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/getamis/sirius/log"
	"github.com/maichain/eth-indexer/common"
	"github.com/maichain/eth-indexer/model"
	"github.com/maichain/eth-indexer/store"
)

var ErrInvalidBalance = errors.New("invalid balance")

type dbAPI struct {
	manager store.ServiceManager
}

// NewAPIWithStore news a account api with store manager
func NewAPIWithStore(store store.ServiceManager) API {
	return &dbAPI{
		manager: store,
	}
}

func (api *dbAPI) GetBalance(ctx context.Context, address gethCommon.Address, blockNr int64) (balance *big.Int, blockNumber *big.Int, err error) {
	logger := log.New("addr", address.Hex(), "number", blockNr)
	// Find state block
	var stateBlock *model.StateBlock
	if common.IsLatestBlock(blockNr) {
		stateBlock, err = api.manager.LastStateBlock()
	} else {
		stateBlock, err = api.manager.FindStateBlock(blockNr)
	}
	// State block should not have not found error
	if err != nil {
		logger.Error("Failed to find state block", "err", err)
		return nil, nil, err
	}
	blockNumber = big.NewInt(stateBlock.Number)

	// Find account
	account, err := api.manager.FindAccount(address, stateBlock.Number)
	if err != nil {
		logger.Error("Failed to find account", "err", err)
		return nil, nil, err
	}
	var ok bool
	balance, ok = new(big.Int).SetString(account.Balance, 10)
	if !ok {
		logger.Error("Failed to covert balance", "balance", account.Balance)
		return nil, nil, ErrBlockNotFound
	}

	return
}
