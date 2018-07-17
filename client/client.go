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

package client

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/getamis/eth-indexer/contracts"
	"github.com/getamis/eth-indexer/model"
	"github.com/getamis/sirius/log"
)

var (
	ErrInvalidTDFormat = errors.New("invalid td format")

	// maxUncles represents maximum number of uncles allowed in a single block
	maxUncles = 2
)

//go:generate mockery -name EthClient

type EthClient interface {
	Balancer

	BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error)
	BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error)
	TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error)
	TransactionByHash(ctx context.Context, hash common.Hash) (tx *types.Transaction, isPending bool, err error)
	UncleByBlockHashAndPosition(ctx context.Context, hash common.Hash, position uint) (*types.Header, error)
	UnclesByBlockHash(ctx context.Context, blockHash common.Hash) ([]*types.Header, error)
	SubscribeNewHead(ctx context.Context, ch chan<- *types.Header) (ethereum.Subscription, error)
	DumpBlock(ctx context.Context, blockNr int64) (*state.Dump, error)
	BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error)
	CallContract(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error)
	ModifiedAccountStatesByNumber(ctx context.Context, num uint64) (*state.DirtyDump, error)
	CodeAt(ctx context.Context, account common.Address, blockNumber *big.Int) ([]byte, error)
	GetERC20(ctx context.Context, addr common.Address, num int64) (*model.ERC20, error)
	GetTotalDifficulty(ctx context.Context, hash common.Hash) (*big.Int, error)
	GetBlockReceipts(ctx context.Context, hash common.Hash) (types.Receipts, error)
	// Get ETH transfer logs
	GetTransferLogs(ctx context.Context, hash common.Hash) ([]*types.TransferLog, error)
	BatchBalanceAt(ctx context.Context, accounts []common.Address, blockNumber *big.Int) ([]*big.Int, error)
	BatchCallContract(ctx context.Context, msgs []*ethereum.CallMsg, blockNumber *big.Int) ([][]byte, error)
	Close()
}

type client struct {
	*ethclient.Client
	rpc *rpc.Client
}

func NewClient(url string) (EthClient, error) {
	rpcClient, err := rpc.Dial(url)
	if err != nil {
		return nil, err
	}
	client := &client{
		Client: ethclient.NewClient(rpcClient),
		rpc:    rpcClient,
	}
	return newCacheMiddleware(client), nil
}

func (c *client) TransactionReceipts(ctx context.Context, txs types.Transactions) ([]*types.Receipt, error) {
	lens := len(txs)
	if lens == 0 {
		// Don't need to process this requests
		return nil, nil
	}

	// Construct batch requests
	method := "eth_getTransactionReceipt"
	var reqs []rpc.BatchElem
	receipts := make([]*types.Receipt, lens)
	for i, tx := range txs {
		receipts[i] = &types.Receipt{}
		reqs = append(reqs, rpc.BatchElem{
			Method: method,
			Args:   []interface{}{tx.Hash()},
			Result: receipts[i],
		})
	}

	// Batch calls
	err := c.rpc.BatchCallContext(ctx, reqs)
	if err != nil {
		return nil, err
	}

	// Ensure all requests are ok
	for _, req := range reqs {
		if req.Error != nil {
			return nil, req.Error
		}
	}

	return receipts, nil
}

func (c *client) UncleByBlockHashAndPosition(ctx context.Context, hash common.Hash, position uint) (*types.Header, error) {
	var result *types.Header
	err := c.rpc.CallContext(ctx, &result, "eth_getUncleByBlockHashAndIndex", hash, hexutil.Uint(position))
	if err == nil && result == nil {
		err = ethereum.NotFound
	}
	return result, err
}

func (c *client) UnclesByBlockHash(ctx context.Context, blockHash common.Hash) ([]*types.Header, error) {
	var result []*types.Header
	for i := 0; i < maxUncles; i++ {
		h, err := c.UncleByBlockHashAndPosition(ctx, blockHash, uint(i))
		if err != ethereum.NotFound {
			return nil, err
		}
		if h == nil {
			continue
		}
		result = append(result, h)
	}
	return result, nil
}

func (c *client) DumpBlock(ctx context.Context, blockNr int64) (*state.Dump, error) {
	r := &state.Dump{}
	err := c.rpc.CallContext(ctx, r, "debug_dumpBlock", fmt.Sprintf("0x%x", blockNr))
	return r, err
}

func (c *client) GetTotalDifficulty(ctx context.Context, hash common.Hash) (*big.Int, error) {
	var r string
	err := c.rpc.CallContext(ctx, &r, "debug_getTotalDifficulty", hash.Hex())
	if err != nil {
		return nil, err
	}
	// Remove the '0x' prefix
	td, ok := new(big.Int).SetString(r[2:], 16)
	if !ok {
		return nil, ErrInvalidTDFormat
	}
	return td, nil
}

func (c *client) GetBlockReceipts(ctx context.Context, hash common.Hash) (types.Receipts, error) {
	var r types.Receipts
	err := c.rpc.CallContext(ctx, &r, "debug_getBlockReceipts", hash.Hex())
	return r, err
}

func (c *client) ModifiedAccountStatesByNumber(ctx context.Context, num uint64) (*state.DirtyDump, error) {
	r := &state.DirtyDump{}
	err := c.rpc.CallContext(ctx, r, "debug_getModifiedAccountStatesByNumber", num)
	return r, err
}

func (c *client) GetERC20(ctx context.Context, addr common.Address, num int64) (*model.ERC20, error) {
	logger := log.New("addr", addr, "number", num)
	code, err := c.CodeAt(ctx, addr, nil)
	if err != nil {
		return nil, err
	}
	erc20 := &model.ERC20{
		Address:     addr.Bytes(),
		Code:        code,
		BlockNumber: num,
	}

	caller, err := contracts.NewERC20TokenCaller(addr, c)
	if err != nil {
		logger.Warn("Failed to initiate contract caller", "err", err)
	} else {
		// Set decimals
		decimal, err := caller.Decimals(&bind.CallOpts{})
		if err != nil {
			logger.Warn("Failed to get decimals", "err", err)
		}
		erc20.Decimals = int(decimal)

		// Set total supply
		supply, err := caller.TotalSupply(&bind.CallOpts{})
		if err != nil {
			logger.Warn("Failed to get total supply", "err", err)
		}
		erc20.TotalSupply = supply.String()

		// Set name
		name, err := caller.Name(&bind.CallOpts{})
		if err != nil {
			logger.Warn("Failed to get name", "err", err)
		}
		erc20.Name = name
	}
	return erc20, nil
}

func (c *client) GetTransferLogs(ctx context.Context, hash common.Hash) ([]*types.TransferLog, error) {
	r := []*types.TransferLog{}
	err := c.rpc.CallContext(ctx, &r, "debug_getTransferLogs", hash.Hex())
	return r, err
}

func (c *client) BatchBalanceAt(ctx context.Context, accounts []common.Address, blockNumber *big.Int) ([]*big.Int, error) {
	lens := len(accounts)
	if lens == 0 {
		return nil, nil
	}

	blockNumStr := hexutil.EncodeBig(blockNumber)
	// Construct batch requests
	method := "eth_getBalance"
	reqs := make([]rpc.BatchElem, lens)
	for i, account := range accounts {
		var result hexutil.Big
		reqs[i] = rpc.BatchElem{
			Method: method,
			Args:   []interface{}{account, blockNumStr},
			Result: &result,
		}
	}

	// Batch calls
	err := c.rpc.BatchCallContext(ctx, reqs)
	if err != nil {
		return nil, err
	}

	// Ensure all requests are ok
	balances := make([]*big.Int, lens)
	for i, req := range reqs {
		if req.Error != nil {
			return nil, req.Error
		}
		balances[i] = (*big.Int)(req.Result.(*hexutil.Big))
	}
	return balances, nil
}

func (c *client) BatchCallContract(ctx context.Context, msgs []*ethereum.CallMsg, blockNumber *big.Int) ([][]byte, error) {
	lens := len(msgs)
	if lens == 0 {
		return nil, nil
	}

	blockNumStr := hexutil.EncodeBig(blockNumber)
	// Construct batch requests
	method := "eth_call"
	reqs := make([]rpc.BatchElem, lens)
	for i, msg := range msgs {
		var hex hexutil.Bytes
		reqs[i] = rpc.BatchElem{
			Method: method,
			Args:   []interface{}{toCallArg(msg), blockNumStr},
			Result: &hex,
		}

	}

	// Batch calls
	err := c.rpc.BatchCallContext(ctx, reqs)
	if err != nil {
		return nil, err
	}

	// Ensure all requests are ok
	results := make([][]byte, lens)
	for i, req := range reqs {
		if req.Error != nil {
			return nil, req.Error
		}
		results[i] = []byte(*req.Result.(*hexutil.Bytes))
	}
	return results, nil
}

func toCallArg(msg *ethereum.CallMsg) interface{} {
	arg := map[string]interface{}{
		"from": msg.From,
		"to":   msg.To,
	}
	if len(msg.Data) > 0 {
		arg["data"] = hexutil.Bytes(msg.Data)
	}
	if msg.Value != nil {
		arg["value"] = (*hexutil.Big)(msg.Value)
	}
	if msg.Gas != 0 {
		arg["gas"] = hexutil.Uint64(msg.Gas)
	}
	if msg.GasPrice != nil {
		arg["gasPrice"] = (*hexutil.Big)(msg.GasPrice)
	}
	return arg
}

func (c *client) Close() {
	c.rpc.Close()
}
