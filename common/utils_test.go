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
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
)

func TestAccumulateRewards(t *testing.T) {
	header := &types.Header{Number: big.NewInt(5862127)}

	tests := []struct {
		description  string
		uncleHeaders []*types.Header
		minerReward  *big.Int
		unclesReward []*big.Int
	}{
		{
			description:  "no uncles",
			uncleHeaders: []*types.Header{},
			minerReward:  big.NewInt(3000000000000000000),
			unclesReward: []*big.Int{},
		},
		{
			description:  "two uncles in same block number",
			uncleHeaders: []*types.Header{{Number: big.NewInt(5862126)}, {Number: big.NewInt(5862126)}},
			minerReward:  big.NewInt(3187500000000000000),
			unclesReward: []*big.Int{big.NewInt(2625000000000000000), big.NewInt(2625000000000000000)},
		},
		{
			description:  "two uncles in different block number",
			uncleHeaders: []*types.Header{{Number: big.NewInt(5862126)}, {Number: big.NewInt(5862125)}},
			minerReward:  big.NewInt(3187500000000000000),
			unclesReward: []*big.Int{big.NewInt(2625000000000000000), big.NewInt(2250000000000000000)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			minerReward, unclesReward := AccumulateRewards(header, tt.uncleHeaders)

			assert.EqualValues(t, len(tt.unclesReward), len(unclesReward))
			for i, u := range tt.unclesReward {
				assert.Zero(t, u.Cmp(unclesReward[i]))
			}
			assert.Zero(t, tt.minerReward.Cmp(minerReward))
		})
	}
}
