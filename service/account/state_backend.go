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

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
)

var (
	// This nil assignment ensures compile time that StateBackend implements bind.ContractBackend.
	_ bind.ContractBackend = (*StateBackend)(nil)

	// ErrNotImplemented returns if the function is not implemented
	ErrNotImplemented = errors.New("not implemented")
)

// StateBackend implements bind.ContractBackend. Its main purpose is to allow using contract bindings at specific block and state.
type StateBackend struct {
	header  *types.Header
	stateDB *state.StateDB
}

// NewStateBackend creates a new binding backend
func NewStateBackend(header *types.Header, stateDB *state.StateDB) *StateBackend {
	return &StateBackend{
		header:  header,
		stateDB: stateDB,
	}
}

// ContractCaller interface

// CodeAt returns the code associated with a certain account in the blockchain.
func (b *StateBackend) CodeAt(ctx context.Context, contract common.Address, blockNumber *big.Int) ([]byte, error) {
	return b.stateDB.GetCode(contract), nil
}

// CallContract executes a contract call.
func (b *StateBackend) CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	// Ensure message is initialized properly.
	if call.GasPrice == nil {
		call.GasPrice = common.Big0
	}
	if call.Gas == 0 {
		call.Gas = 50000000
	}
	if call.Value == nil {
		call.Value = common.Big0
	}

	// Create new call message
	msg := types.NewMessage(call.From, call.To, 0, call.Value, call.Gas, call.GasPrice, call.Data, false)
	evmContext := core.NewEVMContext(msg, b.header, nil, nil)
	// Create a new environment which holds all relevant information
	// about the transaction and calling mechanisms.
	vmenv := vm.NewEVM(evmContext, b.stateDB, nil, vm.Config{})
	gaspool := new(core.GasPool).AddGas(math.MaxUint64)
	r, _, _, err := core.NewStateTransition(vmenv, msg, gaspool).TransitionDb()
	return r, err
}

// ContractTransactor interface

// PendingCodeAt returns the code of the given account in the pending state.
func (b *StateBackend) PendingCodeAt(ctx context.Context, account common.Address) ([]byte, error) {
	return nil, ErrNotImplemented
}

// PendingNonceAt implements PendingStateReader.PendingNonceAt, retrieving
// the nonce currently pending for the account.
func (b *StateBackend) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	return 0, ErrNotImplemented
}

// SuggestGasPrice implements ContractTransactor.SuggestGasPrice. Since the simulated
// chain doens't have miners, we just return a gas price of 1 for any call.
func (b *StateBackend) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	return nil, ErrNotImplemented
}

// EstimateGas executes the requested code against the currently pending block/state and
// returns the used amount of gas.
func (b *StateBackend) EstimateGas(ctx context.Context, call ethereum.CallMsg) (uint64, error) {
	return 0, ErrNotImplemented
}

// SendTransaction updates the pending block to include the given transaction.
// It panics if the transaction is invalid.
func (b *StateBackend) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	return ErrNotImplemented
}

// ContractFilterer interface

// FilterLogs executes a log filter operation, blocking during execution and
// returning all the results in one batch.
func (b *StateBackend) FilterLogs(ctx context.Context, query ethereum.FilterQuery) ([]types.Log, error) {
	return nil, ErrNotImplemented
}

// SubscribeFilterLogs creates a background log filtering operation, returning
// a subscription immediately, which can be used to stream the found events.
func (b *StateBackend) SubscribeFilterLogs(ctx context.Context, query ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error) {
	return nil, ErrNotImplemented
}
