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
	"math/big"

	"github.com/ethereum/go-ethereum"
	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/getamis/eth-indexer/contracts"
	"github.com/getamis/eth-indexer/model"
	"github.com/getamis/sirius/log"
)

const (
	chunkSize = 200
)

//go:generate mockery -name Balancer
// Balancer is a wrapper interface to batch get balances
type Balancer interface {
	// BalanceOf returns the balances of ETH and multiple erc20 tokens for multiple accounts
	BalanceOf(context.Context, *big.Int, map[ethCommon.Address]map[ethCommon.Address]struct{}) (map[ethCommon.Address]map[ethCommon.Address]*big.Int, error)
}

// BalanceOf returns the balances of ETH and multiple erc20 tokens for multiple accounts
func (c *client) BalanceOf(ctx context.Context, blockNumber *big.Int, addrs map[ethCommon.Address]map[ethCommon.Address]struct{}) (balances map[ethCommon.Address]map[ethCommon.Address]*big.Int, err error) {
	logger := log.New("number", blockNumber.Int64())

	var msgs []*ethereum.CallMsg
	var owners []ethCommon.Address
	// Only handle non-ETH balances
	for erc20Addr, list := range addrs {
		if erc20Addr == model.ETHAddress {
			continue
		}
		for addr := range list {
			// Append balance of message
			msgs = append(msgs, contracts.BalanceOfMsg(erc20Addr, addr))
			owners = append(owners, addr)
		}
	}

	balances = make(map[ethCommon.Address]map[ethCommon.Address]*big.Int)

	// Get batch results
	lens := len(msgs)
	for begin := 0; begin < lens; begin += chunkSize {
		end := begin + chunkSize
		if end > lens {
			end = lens
		}

		chunk := msgs[begin:end]
		logger.Info("processing ERC20 balance chunk", "total", lens, "begin", begin, "end", end)
		outputs, err := c.BatchCallContract(ctx, chunk, blockNumber)
		if err != nil {
			return nil, err
		}

		for i := 0; i < len(chunk); i++ {
			balance, err := contracts.DecodeBalanceOf(outputs[i])
			if err != nil {
				return nil, err
			}

			contractAddr := *chunk[i].To
			if balances[contractAddr] == nil {
				balances[contractAddr] = make(map[ethCommon.Address]*big.Int)
			}
			balances[contractAddr][owners[begin+i]] = balance
		}
	}

	// Handle ETH balances
	if _, ok := addrs[model.ETHAddress]; ok {
		balances[model.ETHAddress], err = c.ethBalanceOf(ctx, blockNumber, addrs[model.ETHAddress])
		if err != nil {
			return nil, err
		}
	}
	return
}

// ethBalanceOf returns the ether balances
func (c *client) ethBalanceOf(ctx context.Context, blockNumber *big.Int, addrs map[ethCommon.Address]struct{}) (etherBalances map[ethCommon.Address]*big.Int, err error) {
	logger := log.New("number", blockNumber.Int64())
	lens := len(addrs)
	var addrList []ethCommon.Address
	for addr := range addrs {
		addrList = append(addrList, addr)
	}

	// Construct ether balances
	etherBalances = make(map[ethCommon.Address]*big.Int, lens)

	// Get ethers
	for begin := 0; begin < lens; begin += chunkSize {
		end := begin + chunkSize
		if end > lens {
			end = lens
		}

		chunk := addrList[begin:end]
		logger.Info("processing ETH balance chunk", "total", lens, "begin", begin, "end", end)
		ethers, err := c.BatchBalanceAt(ctx, chunk, blockNumber)
		if err != nil {
			return nil, err
		}

		for i, e := range ethers {
			etherBalances[addrList[begin+i]] = e
		}
	}
	return
}
