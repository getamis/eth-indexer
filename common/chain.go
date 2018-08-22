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
	"errors"

	"github.com/ethereum/go-ethereum/params"
)

// Chain represents the chain type
type Chain int

const (
	// MainChain represents the main chain config
	MainChain Chain = iota
	// TestChain represents the test chain config
	TestChain
	// RopstenChain represents the ropsten chain config
	RopstenChain
)

var ErrUnknownChain = errors.New("unknown chain")

func GetChain(chain Chain) (*params.ChainConfig, error) {
	if chain == MainChain {
		return params.MainnetChainConfig, nil
	}
	if chain == TestChain {
		return params.TestChainConfig, nil
	}
	if chain == RopstenChain {
		return params.TestnetChainConfig, nil
	}
	return nil, ErrUnknownChain
}
