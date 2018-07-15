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
	"bytes"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	gethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/getamis/eth-indexer/common"
	"github.com/getamis/eth-indexer/contracts"
	"github.com/getamis/eth-indexer/model"
)

var (
	erc20ABI, _ = abi.JSON(strings.NewReader(contracts.ERC20TokenABI))
	// The sha3 signature of transfer event in erc20
	// event Transfer(address indexed _from, address indexed _to, uint256 _value)
	sha3TransferEvent = common.HexToBytes("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")
)

func (m *manager) erc20Events(blockNumber int64, txHash gethCommon.Hash, logs []*model.Log) (events []*model.Transfer, err error) {
	// Parse the logs
	for _, l := range logs {
		// Insert transfer event if the contract address is in our erc20 list and it's a transfer event
		addr := gethCommon.BytesToAddress(l.ContractAddress)
		if _, ok := m.tokenList[addr]; ok && bytes.Equal(l.EventName, sha3TransferEvent) {
			// Get the tranfer event
			event := &contracts.ERC20TokenTransfer{}
			err := erc20ABI.Unpack(event, "Transfer", l.Data)
			if err != nil {
				return nil, err
			}

			// Convert to model.ERC20Transfer
			events = append(events, &model.Transfer{
				Address:     l.ContractAddress,
				BlockNumber: blockNumber,
				TxHash:      txHash.Bytes(),
				From:        gethCommon.BytesToAddress(l.Topic1).Bytes(),
				To:          gethCommon.BytesToAddress(l.Topic2).Bytes(),
				Value:       event.Value.String(),
			})
		}
	}
	return
}
