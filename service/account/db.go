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
	"math"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	gethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/getamis/sirius/log"
	"github.com/jinzhu/gorm"
	"github.com/maichain/eth-indexer/common"
	"github.com/maichain/eth-indexer/model"
	"github.com/maichain/eth-indexer/service/account/contracts"
	accountStore "github.com/maichain/eth-indexer/store/account"
)

var ErrInvalidBalance = errors.New("invalid balance")

type dbAPI struct {
	store accountStore.Store
}

// NewAPIWithWithDB news a account api with DB
func NewAPIWithWithDB(db *gorm.DB) API {
	return &dbAPI{
		store: accountStore.NewWithDB(db),
	}
}

func (api *dbAPI) GetBalance(ctx context.Context, address gethCommon.Address, blockNr int64) (balance *big.Int, blockNumber *big.Int, err error) {
	logger := log.New("addr", address.Hex(), "number", blockNr)
	// Find state block
	var stateBlock *model.StateBlock
	if common.QueryLatestBlock(blockNr) {
		stateBlock, err = api.store.LastStateBlock()
	} else {
		stateBlock, err = api.store.FindStateBlock(blockNr)
	}
	// State block should not have not found error
	if err != nil {
		logger.Error("Failed to find state block", "err", err)
		return nil, nil, err
	}
	blockNumber = big.NewInt(stateBlock.Number)

	// Find account
	account, err := api.store.FindAccount(address, stateBlock.Number)
	if err != nil {
		logger.Error("Failed to find account", "err", err)
		return nil, nil, err
	} else {
		var ok bool
		balance, ok = new(big.Int).SetString(account.Balance, 10)
		if !ok {
			logger.Error("Failed to covert balance", "balance", account.Balance)
			return nil, nil, ErrBlockNotFound
		}
	}

	return
}

// TODO: Not verified yet
func (api *dbAPI) GetERC20Balance(ctx context.Context, contractAddress, address gethCommon.Address, blockNr int64) (balance *big.Int, blockNumber *big.Int, err error) {
	logger := log.New("contractAddr", contractAddress.Hex(), "addr", address.Hex(), "number", blockNr)
	// Find contract code
	contractCode, err := api.store.FindContractCode(contractAddress)
	if err != nil {
		logger.Error("Failed to find contract code", "err", err)
		return nil, nil, err
	}

	// Find state block
	var stateBlock *model.StateBlock
	if common.QueryLatestBlock(blockNr) {
		stateBlock, err = api.store.LastStateBlock()
	} else {
		stateBlock, err = api.store.FindStateBlock(blockNr)
	}
	// State block should not have not found error
	if err != nil {
		logger.Error("Failed to find state block", "err", err)
		return nil, nil, err
	}
	blockNumber = big.NewInt(stateBlock.Number)

	// Find contract
	contract, err := api.store.FindContract(contractAddress, stateBlock.Number)
	if err != nil {
		logger.Error("Failed to find contract", "err", err)
		return nil, nil, err
	}

	// Get balance from contract
	balance, err = getBalance(contractAddress, address, contractCode, contract)
	if err != nil {
		logger.Error("Failed to get balance", "err", err)
		return nil, nil, err
	}
	return
}

func getBalance(contractAddress, address gethCommon.Address, code *model.ContractCode, contractData *model.Contract) (balance *big.Int, err error) {
	from := &account{}
	to := &account{
		address: contractAddress,
	}

	// Construct EVM and contract
	contract := vm.NewContract(from, to, gethCommon.Big0, math.MaxUint64)
	contract.SetCallCode(&contractAddress, gethCommon.BytesToHash(code.Hash), gethCommon.Hex2Bytes(code.Code))
	evm := vm.NewEVM(vm.Context{}, nil, params.MainnetChainConfig, vm.Config{})
	inter := vm.NewInterpreter(evm, vm.Config{})

	// Create new call message
	parsed, err := abi.JSON(strings.NewReader(contracts.ERC20ABI))
	method := "balanceOf"
	data, err := parsed.Pack(method)
	if err != nil {
		log.Error("Failed to parse balanceOf method", "err", err)
		return
	}

	// Run contract
	ret, err := inter.Run(contract, data)
	if err != nil {
		log.Error("Failed to run contract", "err", err)
		return
	}

	// Unpack result into balance
	err = parsed.Unpack(balance, method, ret)
	return
}
