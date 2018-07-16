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

package model

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

func TestHeader_AddReward(t *testing.T) {
	tests := []struct {
		Description          string
		TxsFee               *big.Int
		MinerBaseReward      *big.Int
		UncleInclusionReward *big.Int
		UnclesReward         []*big.Int
		UncleCBs             []common.Address
		UnclesHash           []common.Hash

		err error
	}{
		{
			Description:          "no uncle",
			TxsFee:               big.NewInt(10),
			MinerBaseReward:      big.NewInt(10),
			UncleInclusionReward: big.NewInt(0),
			UnclesReward:         nil,
			UncleCBs:             nil,
			UnclesHash:           nil,

			err: nil,
		},
		{
			Description:          "1 uncle",
			TxsFee:               big.NewInt(10),
			MinerBaseReward:      big.NewInt(10),
			UncleInclusionReward: big.NewInt(5),
			UnclesReward:         []*big.Int{big.NewInt(3)},
			UncleCBs:             []common.Address{common.BytesToAddress([]byte("cb1"))},
			UnclesHash:           []common.Hash{common.BytesToHash([]byte("cb1"))},

			err: nil,
		},
		{
			Description:          "2 uncles",
			TxsFee:               big.NewInt(10),
			MinerBaseReward:      big.NewInt(10),
			UncleInclusionReward: big.NewInt(5),
			UnclesReward:         []*big.Int{big.NewInt(3), big.NewInt(4)},
			UncleCBs:             []common.Address{common.BytesToAddress([]byte("cb1")), common.BytesToAddress([]byte("cb2"))},
			UnclesHash:           []common.Hash{common.BytesToHash([]byte("cb1")), common.BytesToHash([]byte("cb2"))},

			err: nil,
		},

		{
			Description:          "incorrect uncle numbers",
			TxsFee:               big.NewInt(10),
			MinerBaseReward:      big.NewInt(10),
			UncleInclusionReward: big.NewInt(5),
			UnclesReward:         []*big.Int{big.NewInt(3)},
			UncleCBs:             []common.Address{common.BytesToAddress([]byte("cb1")), common.BytesToAddress([]byte("cb2"))},
			UnclesHash:           []common.Hash{common.BytesToHash([]byte("cb1")), common.BytesToHash([]byte("cb2"))},

			err: ErrConfusedUncles,
		},

		{
			Description:          "too many uncles",
			TxsFee:               big.NewInt(10),
			MinerBaseReward:      big.NewInt(10),
			UncleInclusionReward: big.NewInt(5),
			UnclesReward:         []*big.Int{big.NewInt(3), big.NewInt(3), big.NewInt(3)},
			UncleCBs:             []common.Address{common.BytesToAddress([]byte("cb1")), common.BytesToAddress([]byte("cb2")), common.BytesToAddress([]byte("cb3"))},
			UnclesHash:           []common.Hash{common.BytesToHash([]byte("cb1")), common.BytesToHash([]byte("cb2")), common.BytesToHash([]byte("cb3"))},

			err: ErrTooManyUncles,
		},
	}

	for _, tt := range tests {
		t.Run(tt.Description, func(t *testing.T) {
			h := Header{}
			newH, err := h.AddReward(
				tt.TxsFee,
				tt.MinerBaseReward,
				tt.UncleInclusionReward,
				tt.UnclesReward,
				tt.UncleCBs,
				tt.UnclesHash,
			)

			// returned if get error
			if err != nil {
				assert.Nil(t, newH)
				assert.Equal(t, tt.err, err)
				return
			}

			assert.NoError(t, err)
			switch len(tt.UnclesHash) {
			case 0:
				assert.Equal(t, []byte{}, newH.Uncle1Coinbase)
				assert.Equal(t, []byte{}, newH.Uncle2Coinbase)

				assert.Equal(t, []byte{}, newH.Uncle1Hash)
				assert.Equal(t, []byte{}, newH.Uncle2Hash)

				assert.Equal(t, "0", newH.Uncle1Reward)
				assert.Equal(t, "0", newH.Uncle2Reward)
			case 1:
				assert.Equal(t, tt.UncleCBs[0].Bytes(), newH.Uncle1Coinbase)
				assert.Equal(t, []byte{}, newH.Uncle2Coinbase)

				assert.Equal(t, tt.UnclesHash[0].Bytes(), newH.Uncle1Hash)
				assert.Equal(t, []byte{}, newH.Uncle2Hash)

				assert.Equal(t, tt.UnclesReward[0].String(), newH.Uncle1Reward)
				assert.Equal(t, "0", newH.Uncle2Reward)
			case 2:
				assert.Equal(t, tt.UncleCBs[0].Bytes(), newH.Uncle1Coinbase)
				assert.Equal(t, tt.UncleCBs[1].Bytes(), newH.Uncle2Coinbase)

				assert.Equal(t, tt.UnclesHash[0].Bytes(), newH.Uncle1Hash)
				assert.Equal(t, tt.UnclesHash[1].Bytes(), newH.Uncle2Hash)

				assert.Equal(t, tt.UnclesReward[0].String(), newH.Uncle1Reward)
				assert.Equal(t, tt.UnclesReward[1].String(), newH.Uncle2Reward)
			}
		})
	}
}
