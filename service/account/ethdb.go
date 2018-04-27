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

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	gethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/getamis/sirius/log"
	"github.com/maichain/eth-indexer/common"
	"github.com/maichain/eth-indexer/service/account/contracts"
)

// missingNumber is returned by GetBlockNumber if no header with the
// given block hash has been stored in the database
const missingNumber = uint64(0xffffffffffffffff)

var (
	// Key for last head
	headHeaderKey = []byte("LastHeader")
	// Key for last block
	headBlockKey = []byte("LastBlock")
	// Key for last fast
	headFastKey = []byte("LastFast")
	// ErrBlockNotFound returns if the block is not found
	ErrBlockNotFound = errors.New("block not found")
)

type ethDBAPI struct {
	ethDB   ethdb.Database
	stateDB state.Database
}

// NewAPIWithWithEthDB news a account api with eth DB
func NewAPIWithWithEthDB(db ethdb.Database) API {
	return &ethDBAPI{
		ethDB:   db,
		stateDB: state.NewDatabase(db),
	}
}

func (api *ethDBAPI) GetBalance(ctx context.Context, address gethCommon.Address, blockNr int64) (*big.Int, *big.Int, error) {
	state, header, err := api.stateAt(ctx, blockNr)
	if err != nil {
		log.Error("Failed to get ETH balance", "addr", address.Hex(), "number", blockNr, "err", err)
		return nil, nil, err
	}
	b := state.GetBalance(address)
	return header.Number, b, state.Error()
}

// TODO: Not verified yet
func (api *ethDBAPI) GetERC20Balance(ctx context.Context, contractAddress, address gethCommon.Address, blockNr int64) (*big.Int, *big.Int, error) {
	state, header, err := api.stateAt(ctx, blockNr)
	if err != nil {
		return nil, nil, err
	}

	backend := NewStateBackend(header, state)
	erc20, err := contracts.NewERC20(contractAddress, backend)
	if err != nil {
		return nil, nil, err
	}

	balance, err := erc20.BalanceOf(&bind.CallOpts{}, address)
	if err != nil {
		return nil, nil, err
	}
	return header.Number, balance, nil
}

func (api *ethDBAPI) stateAt(ctx context.Context, blockNr int64) (*state.StateDB, *types.Header, error) {
	// Get header
	header, err := api.headerByNumber(ctx, blockNr)
	if err != nil {
		log.Error("Failed to get header by number", "number", blockNr, "err", err)
		return nil, nil, err
	}

	// Get state
	s, err := state.New(header.Root, api.stateDB)
	if err != nil {
		log.Error("Failed to new state by header", "header", header.Hash(), "number", blockNr, "err", err)
		return nil, nil, err
	}
	return s, header, err
}

func (api *ethDBAPI) headerByNumber(ctx context.Context, blockNr int64) (*types.Header, error) {
	var hash gethCommon.Hash
	var number uint64
	if common.IsLatestBlock(blockNr) {
		hash = core.GetHeadBlockHash(api.ethDB)
		if hash == (gethCommon.Hash{}) {
			return nil, ErrBlockNotFound
		}
		number = core.GetBlockNumber(api.ethDB, hash)
		if number == missingNumber {
			return nil, ErrBlockNotFound
		}
	} else {
		hash = core.GetCanonicalHash(api.ethDB, uint64(blockNr))
		if hash == (gethCommon.Hash{}) {
			return nil, ErrBlockNotFound
		}
		number = uint64(blockNr)
	}

	header := core.GetHeader(api.ethDB, hash, uint64(blockNr))
	if header == nil {
		return nil, ErrBlockNotFound
	}
	return header, nil
}
