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
	"math"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/getamis/sirius/log"
	"github.com/maichain/eth-indexer/service/account/contracts"
)

// Call calls the specific contract method call in given state
func Call(db vm.StateDB, contractABI string, contractAddress ethCommon.Address, method string, result interface{}, inputs ...interface{}) error {
	// Construct EVM and contract
	contract := vm.NewContract(&account{}, &account{
		address: contractAddress,
	}, ethCommon.Big0, math.MaxUint64)
	contract.SetCallCode(&contractAddress, db.GetCodeHash(contractAddress), db.GetCode(contractAddress))
	evm := vm.NewEVM(vm.Context{}, db, params.MainnetChainConfig, vm.Config{})
	inter := vm.NewInterpreter(evm, vm.Config{})

	// Create new call message
	parsed, err := abi.JSON(strings.NewReader(contractABI))
	data, err := parsed.Pack(method, inputs...)
	if err != nil {
		log.Error("Failed to parse balanceOf method", "err", err)
		return err
	}

	// Run contract
	ret, err := inter.Run(contract, data)
	if err != nil {
		log.Error("Failed to run contract", "err", err)
		return err
	}

	// Unpack result into result
	return parsed.Unpack(result, method, ret)
}

// BalanceOf returns the amount of ERC20 token at the given state db
func BalanceOf(db vm.StateDB, contractAddress ethCommon.Address, address ethCommon.Address) (*big.Int, error) {
	result := new(*big.Int)
	err := Call(db, contracts.ERC20TokenABI, contractAddress, "balanceOf", result, address)
	if err != nil {
		return nil, err
	}
	return *result, nil
}
