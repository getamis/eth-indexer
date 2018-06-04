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
package store

import (
	"context"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/getamis/eth-indexer/model"
	acctMock "github.com/getamis/eth-indexer/store/account/mocks"
	hdrMock "github.com/getamis/eth-indexer/store/block_header/mocks"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("DB Eth Balance Test", func() {
	var mockAccountStore *acctMock.Store
	var mockHdrStore *hdrMock.Store
	var manager *serviceManager
	var addr common.Address
	blockNumber := int64(10)
	header := &model.Header{
		Number: 100,
	}

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

	Context("with valid parameters", func() {
		account := &model.Account{
			Address: addr.Bytes(),
			Balance: "1000",
		}
		accountBalance, _ := new(big.Int).SetString(account.Balance, 10)
		It("latest block", func() {
			mockHdrStore.On("FindLatestBlock").Return(header, nil).Once()
			mockAccountStore.On("FindAccount", addr, header.Number).Return(account, nil).Once()
			expBalance, expNumber, err := manager.GetBalance(context.Background(), addr, -1)
			Expect(err).Should(BeNil())
			Expect(expBalance).Should(Equal(accountBalance))
			Expect(expNumber.Int64()).Should(Equal(header.Number))
		})
		It("certain block", func() {
			mockHdrStore.On("FindBlockByNumber", blockNumber).Return(header, nil).Once()
			mockAccountStore.On("FindAccount", addr, header.Number).Return(account, nil).Once()
			expBalance, expNumber, err := manager.GetBalance(context.Background(), addr, blockNumber)
			Expect(err).Should(BeNil())
			Expect(expBalance).Should(Equal(accountBalance))
			Expect(expNumber.Int64()).Should(Equal(header.Number))
		})
	})

	Context("with invalid parameters", func() {
		unknownErr := errors.New("unknown error")
		It("failed to find state block", func() {
			mockHdrStore.On("FindBlockByNumber", blockNumber).Return(nil, unknownErr).Once()
			expBalance, expNumber, err := manager.GetBalance(context.Background(), addr, blockNumber)
			Expect(err).Should(Equal(unknownErr))
			Expect(expBalance).Should(BeNil())
			Expect(expNumber).Should(BeNil())
		})
		It("failed to find latest state block", func() {
			mockHdrStore.On("FindLatestBlock").Return(nil, unknownErr).Once()
			expBalance, expNumber, err := manager.GetBalance(context.Background(), addr, -1)
			Expect(err).Should(Equal(unknownErr))
			Expect(expBalance).Should(BeNil())
			Expect(expNumber).Should(BeNil())
		})
		It("failed to find account", func() {
			mockHdrStore.On("FindBlockByNumber", blockNumber).Return(header, nil).Once()
			mockAccountStore.On("FindAccount", addr, header.Number).Return(nil, unknownErr).Once()
			expBalance, expNumber, err := manager.GetBalance(context.Background(), addr, blockNumber)
			Expect(err).Should(Equal(unknownErr))
			Expect(expBalance).Should(BeNil())
			Expect(expNumber).Should(BeNil())
		})
	})
})
