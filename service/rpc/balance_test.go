// Copyright 2018 AMIS Technologies
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package rpc

import (
	"context"
	"reflect"

	"database/sql/driver"
	gethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/jinzhu/gorm"
	"github.com/maichain/eth-indexer/service/pb"
	"github.com/maichain/eth-indexer/store/mocks"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"math/big"
)

var _ = Describe("Server Balance Test", func() {
	var (
		mockServiceManager *mocks.ServiceManager
		svr                *server
	)

	BeforeEach(func() {
		mockServiceManager = new(mocks.ServiceManager)
		svr = New(mockServiceManager)
	})

	AfterEach(func() {
		mockServiceManager.AssertExpectations(GinkgoT())
	})

	Context("GetBalance()", func() {
		ctx := context.Background()

		Context("bad block number", func() {
			It("returns error", func() {
				req := &pb.GetBalanceRequest{BlockNumber: -2}
				res, err := svr.GetBalance(ctx, req)
				Expect(err).Should(BeEquivalentTo(ErrInvalidBlockNumber))
				Expect(res).Should(BeNil())
			})
		})

		Context("ETH", func() {
			blockNum := int64(5430100)
			req := &pb.GetBalanceRequest{
				Token:       ethToken,
				Address:     "0x343c43a37d37dff08ae8c4a11544c718abb4fcf8",
				BlockNumber: blockNum + 10}

			Context("account exists", func() {
				It("returns the balance", func() {
					balanceString := "987654321098765432109876543210"
					balance, ok := new(big.Int).SetString(balanceString, 10)
					Expect(ok).Should(BeTrue())
					mockServiceManager.On("GetBalance", ctx, gethCommon.HexToAddress(req.Address), req.BlockNumber).Return(balance, new(big.Int).SetInt64(blockNum), nil).Once()
					res, err := svr.GetBalance(ctx, req)
					Expect(err).Should(Succeed())
					Expect(reflect.DeepEqual(*res, pb.GetBalanceResponse{Amount: balanceString, BlockNumber: blockNum})).Should(BeTrue())
				})
			})

			Context("account does not exist", func() {
				It("returns the balance", func() {
					mockServiceManager.On("GetBalance", ctx, gethCommon.HexToAddress(req.Address), req.BlockNumber).Return(nil, nil, gorm.ErrRecordNotFound).Once()
					res, err := svr.GetBalance(ctx, req)
					Expect(err).ShouldNot(BeNil())
					Expect(res).Should(BeNil())
				})
			})

			Context("transient error", func() {
				It("returns the balance", func() {
					mockServiceManager.On("GetBalance", ctx, gethCommon.HexToAddress(req.Address), req.BlockNumber).Return(nil, nil, driver.ErrBadConn).Once()
					res, err := svr.GetBalance(ctx, req)
					Expect(err).ShouldNot(BeNil())
					Expect(res).Should(BeNil())
				})
			})
		})

		Context("ERC20", func() {
			blockNum := int64(5430100)
			req := &pb.GetBalanceRequest{
				Token:       "0xfffd933a0bc612844eaf0c6fe3e5b8e9b6c1d19c",
				Address:     "0x343c43a37d37dff08ae8c4a11544c718abb4fcf8",
				BlockNumber: blockNum + 10}

			Context("account exists", func() {
				It("returns the balance", func() {
					balanceString := "987654321098765432109876543210"
					balance, ok := new(big.Int).SetString(balanceString, 10)
					Expect(ok).Should(BeTrue())
					mockServiceManager.On("GetERC20Balance", ctx, gethCommon.HexToAddress(req.Token), gethCommon.HexToAddress(req.Address), req.BlockNumber).Return(balance, new(big.Int).SetInt64(blockNum), nil).Once()
					res, err := svr.GetBalance(ctx, req)
					Expect(err).Should(Succeed())
					Expect(reflect.DeepEqual(*res, pb.GetBalanceResponse{Amount: balanceString, BlockNumber: blockNum})).Should(BeTrue())
				})
			})

			Context("account does not exist", func() {
				It("returns the balance", func() {
					mockServiceManager.On("GetERC20Balance", ctx, gethCommon.HexToAddress(req.Token), gethCommon.HexToAddress(req.Address), req.BlockNumber).Return(nil, nil, gorm.ErrRecordNotFound).Once()
					res, err := svr.GetBalance(ctx, req)
					Expect(err).ShouldNot(BeNil())
					Expect(res).Should(BeNil())
				})
			})

			Context("transient error", func() {
				It("returns the balance", func() {
					mockServiceManager.On("GetERC20Balance", ctx, gethCommon.HexToAddress(req.Token), gethCommon.HexToAddress(req.Address), req.BlockNumber).Return(nil, nil, driver.ErrBadConn).Once()
					res, err := svr.GetBalance(ctx, req)
					Expect(err).ShouldNot(BeNil())
					Expect(res).Should(BeNil())
				})
			})
		})
	})
})
