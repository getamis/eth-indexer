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
package erc20

import (
	"context"
	"errors"
	"testing"

	ethereum "github.com/ethereum/go-ethereum"
	ethCommon "github.com/ethereum/go-ethereum/common"
	clientMocks "github.com/getamis/eth-indexer/client/mocks"
	"github.com/getamis/eth-indexer/model"
	. "github.com/getamis/eth-indexer/service"
	"github.com/getamis/eth-indexer/service/pb"
	storeMocks "github.com/getamis/eth-indexer/store/mocks"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ERC20 Test", func() {
	var (
		mockStore  *storeMocks.Manager
		mockClient *clientMocks.EthClient
		srv        *server
	)
	BeforeEach(func() {
		mockStore = new(storeMocks.Manager)
		mockClient = new(clientMocks.EthClient)
		srv = New(mockStore, mockClient)
	})
	AfterEach(func() {
		mockStore.AssertExpectations(GinkgoT())
		mockClient.AssertExpectations(GinkgoT())
	})

	Context("AddERC20()", func() {
		ctx := context.Background()
		unknownErr := errors.New("unknown error")

		It("with valid parameters", func() {
			addr := ethCommon.HexToAddress("0x01")
			req := &pb.AddERC20Request{
				Address:     addr.Hex(),
				BlockNumber: 100,
			}
			erc20 := &model.ERC20{
				BlockNumber: req.BlockNumber,
				Address:     addr.Bytes(),
				Code:        []byte("1234567890"),
				Name:        "name",
				Decimals:    18,
				TotalSupply: "123",
			}
			expRes := &pb.AddERC20Response{
				Address:     req.Address,
				BlockNumber: req.BlockNumber,
				TotalSupply: erc20.TotalSupply,
				Name:        erc20.Name,
				Decimals:    int64(erc20.Decimals),
			}
			mockClient.On("GetERC20", ctx, addr, req.BlockNumber).Return(erc20, nil).Once()
			mockStore.On("InsertERC20", erc20).Return(nil).Once()
			res, err := srv.AddERC20(ctx, req)
			Expect(err).Should(BeNil())
			Expect(res).Should(Equal(expRes))
		})
		Context("with invalid parameters", func() {
			It("failed to insert to db", func() {
				addr := ethCommon.HexToAddress("0x01")
				req := &pb.AddERC20Request{
					Address:     addr.Hex(),
					BlockNumber: 100,
				}
				erc20 := &model.ERC20{
					BlockNumber: req.BlockNumber,
					Address:     addr.Bytes(),
					Code:        []byte("1234567890"),
					Name:        "name",
					Decimals:    18,
					TotalSupply: "123",
				}
				mockClient.On("GetERC20", ctx, addr, req.BlockNumber).Return(erc20, nil).Once()
				mockStore.On("InsertERC20", erc20).Return(unknownErr).Once()
				res, err := srv.AddERC20(ctx, req)
				Expect(err).Should(Equal(ErrInternal))
				Expect(res).Should(BeNil())
			})
			It("failed to get ERC20 code due to unknown error", func() {
				addr := ethCommon.HexToAddress("0x01")
				req := &pb.AddERC20Request{
					Address:     addr.Hex(),
					BlockNumber: 100,
				}
				mockClient.On("GetERC20", ctx, addr, req.BlockNumber).Return(nil, unknownErr).Once()
				res, err := srv.AddERC20(ctx, req)
				Expect(err).Should(Equal(ErrInternal))
				Expect(res).Should(BeNil())
			})
			It("failed to get code due to ethereum not found error", func() {
				addr := ethCommon.HexToAddress("0x01")
				req := &pb.AddERC20Request{
					Address:     addr.Hex(),
					BlockNumber: 100,
				}
				mockClient.On("GetERC20", ctx, addr, req.BlockNumber).Return(nil, ethereum.NotFound).Once()
				res, err := srv.AddERC20(ctx, req)
				Expect(err).Should(Equal(ErrInvalidAddress))
				Expect(res).Should(BeNil())
			})
		})
	})
})

func TestRpcServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ERC20 RPC Test")
}
