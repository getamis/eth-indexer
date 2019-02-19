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

package common

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/assert"
)

func TestAccumulateRewards(t *testing.T) {
	byzantiumBlock := big.NewInt(5862127)
	constantinopleBlock := big.NewInt(7280000)
	tests := []struct {
		description          string
		uncleHeaders         []*types.Header
		blockNum             *big.Int
		uncleInclusionReward *big.Int
		minerBaseReward      *big.Int
		unclesReward         []*big.Int
	}{
		{
			description:          "no uncles on byzantium",
			uncleHeaders:         []*types.Header{},
			blockNum:             byzantiumBlock,
			uncleInclusionReward: big.NewInt(0),
			minerBaseReward:      ethash.ByzantiumBlockReward,
			unclesReward:         []*big.Int{},
		},
		{
			description:          "two uncles in same block number on byzantium",
			uncleHeaders:         []*types.Header{{Number: big.NewInt(byzantiumBlock.Int64() - 1), Coinbase: common.HexToAddress("uncle1")}, {Number: big.NewInt(byzantiumBlock.Int64() - 1), Coinbase: common.HexToAddress("uncle2")}},
			blockNum:             byzantiumBlock,
			uncleInclusionReward: big.NewInt(187500000000000000),
			minerBaseReward:      ethash.ByzantiumBlockReward,
			unclesReward:         []*big.Int{big.NewInt(2625000000000000000), big.NewInt(2625000000000000000)},
		},
		{
			description:          "two uncles in different block number on byzantium",
			uncleHeaders:         []*types.Header{{Number: big.NewInt(byzantiumBlock.Int64() - 1), Coinbase: common.HexToAddress("uncle1")}, {Number: big.NewInt(byzantiumBlock.Int64() - 2), Coinbase: common.HexToAddress("uncle2")}},
			blockNum:             byzantiumBlock,
			uncleInclusionReward: big.NewInt(187500000000000000),
			minerBaseReward:      ethash.ByzantiumBlockReward,
			unclesReward:         []*big.Int{big.NewInt(2625000000000000000), big.NewInt(2250000000000000000)},
		},
		{
			description:          "no uncles on constantinople",
			uncleHeaders:         []*types.Header{},
			blockNum:             constantinopleBlock,
			uncleInclusionReward: big.NewInt(0),
			minerBaseReward:      ethash.ConstantinopleBlockReward,
			unclesReward:         []*big.Int{},
		},
		{
			description:          "two uncles in same block number on constantinople",
			uncleHeaders:         []*types.Header{{Number: big.NewInt(constantinopleBlock.Int64() - 1), Coinbase: common.HexToAddress("uncle1")}, {Number: big.NewInt(constantinopleBlock.Int64() - 1), Coinbase: common.HexToAddress("uncle2")}},
			blockNum:             constantinopleBlock,
			uncleInclusionReward: big.NewInt(125000000000000000),
			minerBaseReward:      ethash.ConstantinopleBlockReward,
			unclesReward:         []*big.Int{big.NewInt(1750000000000000000), big.NewInt(1750000000000000000)},
		},
		{
			description:          "two uncles in different block number on constantinople",
			uncleHeaders:         []*types.Header{{Number: big.NewInt(constantinopleBlock.Int64() - 1), Coinbase: common.HexToAddress("uncle1")}, {Number: big.NewInt(constantinopleBlock.Int64() - 2), Coinbase: common.HexToAddress("uncle2")}},
			blockNum:             constantinopleBlock,
			uncleInclusionReward: big.NewInt(125000000000000000),
			minerBaseReward:      ethash.ConstantinopleBlockReward,
			unclesReward:         []*big.Int{big.NewInt(1750000000000000000), big.NewInt(1500000000000000000)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			header := &types.Header{Number: tt.blockNum}
			minerBaseReward, uncleInclusionReward, unclesCoinbase, unclesReward, _ := AccumulateRewards(params.MainnetChainConfig, header, tt.uncleHeaders)

			assert.Equal(t, tt.minerBaseReward, minerBaseReward)
			assert.EqualValues(t, len(tt.unclesReward), len(unclesReward))
			for i, u := range tt.unclesReward {
				assert.Zero(t, u.Cmp(unclesReward[i]))
			}
			for i, u := range tt.uncleHeaders {
				assert.True(t, bytes.Equal(u.Coinbase.Bytes(), unclesCoinbase[i].Bytes()))
			}
			assert.Zero(t, tt.uncleInclusionReward.Cmp(uncleInclusionReward))
		})
	}
}
