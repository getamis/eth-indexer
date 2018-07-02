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
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/getamis/eth-indexer/model"
	acctMock "github.com/getamis/eth-indexer/store/account/mocks"
	hdrMock "github.com/getamis/eth-indexer/store/block_header/mocks"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("DB Balance Test", func() {
	var mockAccountStore *acctMock.Store
	var mockHdrStore *hdrMock.Store
	var manager *serviceManager
	var addr common.Address
	blockNumber := int64(10)

	BeforeEach(func() {
		mockAccountStore = new(acctMock.Store)
		mockHdrStore = new(hdrMock.Store)
		manager = &serviceManager{
			accountStore:     mockAccountStore,
			blockHeaderStore: mockHdrStore,
		}
		addr = common.HexToAddress(getFakeAddress())
	})

	AfterEach(func() {
		mockAccountStore.AssertExpectations(GinkgoT())
	})

	Context("GetBalance()", func() {
		account := &model.Account{
			Address: addr.Bytes(),
			Balance: "1000",
		}

		It("with valid parameters", func() {
			mockAccountStore.On("FindAccount", model.ETHAddress, addr, blockNumber).Return(account, nil).Once()
			expBalance, expNumber, err := manager.GetBalance(context.Background(), addr, blockNumber)
			Expect(err).Should(BeNil())
			Expect(expBalance.String()).Should(Equal(account.Balance))
			Expect(expNumber.Int64()).Should(Equal(blockNumber))
		})

		Context("with invalid parameters", func() {
			unknownErr := errors.New("unknown error")
			It("failed to find account", func() {
				mockAccountStore.On("FindAccount", model.ETHAddress, addr, blockNumber).Return(account, unknownErr).Once()
				expBalance, expNumber, err := manager.GetBalance(context.Background(), addr, blockNumber)
				Expect(err).Should(Equal(unknownErr))
				Expect(expBalance).Should(BeNil())
				Expect(expNumber).Should(BeNil())
			})
		})
	})

	Context("GetERC20Balance()", func() {
		contractAddress := common.HexToAddress("0x01234567890")
		erc20 := &model.ERC20{
			Name:     "test",
			Address:  contractAddress.Bytes(),
			Decimals: 18,
		}
		account := &model.Account{
			ContractAddress: contractAddress.Bytes(),
			Address:         addr.Bytes(),
			Balance:         "1000000000000000000",
		}

		It("with valid parameters", func() {
			mockAccountStore.On("FindERC20", contractAddress).Return(erc20, nil).Once()
			mockAccountStore.On("FindAccount", contractAddress, addr, blockNumber).Return(account, nil).Once()
			expBalance, expNumber, err := manager.GetERC20Balance(context.Background(), contractAddress, addr, blockNumber)
			Expect(err).Should(BeNil())
			Expect(expBalance.String()).Should(Equal("1"))
			Expect(expNumber.Int64()).Should(Equal(blockNumber))
		})

		Context("with invalid parameters", func() {
			unknownErr := errors.New("unknown error")
			It("failed to find accounts", func() {
				mockAccountStore.On("FindERC20", contractAddress).Return(erc20, nil).Once()
				mockAccountStore.On("FindAccount", contractAddress, addr, blockNumber).Return(nil, unknownErr).Once()
				expBalance, expNumber, err := manager.GetERC20Balance(context.Background(), contractAddress, addr, blockNumber)
				Expect(err).Should(Equal(unknownErr))
				Expect(expBalance).Should(BeNil())
				Expect(expNumber).Should(BeNil())
			})
			It("failed to find erc20", func() {
				mockAccountStore.On("FindERC20", contractAddress).Return(erc20, unknownErr).Once()
				expBalance, expNumber, err := manager.GetERC20Balance(context.Background(), contractAddress, addr, blockNumber)
				Expect(err).Should(Equal(unknownErr))
				Expect(expBalance).Should(BeNil())
				Expect(expNumber).Should(BeNil())
			})
		})
	})
})
