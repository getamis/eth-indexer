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

package store

import (
	"math"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/getamis/eth-indexer/contracts"
	"github.com/getamis/sirius/log"
)

// Call calls the specific contract method call in given state
func Call(db vm.StateDB, contractABI string, contractAddress ethCommon.Address, method string, result interface{}, inputs ...interface{}) error {
	// Construct EVM and contract
	contract := vm.NewContract(&contractAccount{}, &contractAccount{
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
