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
	"math/big"
	"testing"

	ethereum "github.com/ethereum/go-ethereum"
	ethCommon "github.com/ethereum/go-ethereum/common"
	. "github.com/maichain/eth-indexer/service"
	clientMocks "github.com/maichain/eth-indexer/service/indexer/mocks"
	"github.com/maichain/eth-indexer/service/pb"
	storeMocks "github.com/maichain/eth-indexer/store/mocks"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
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
			blockNumber := big.NewInt(100)
			code := []byte("1234567890")
			req := &pb.AddERC20Request{
				Address:     addr.Hex(),
				BlockNumber: blockNumber.Int64(),
			}
			mockClient.On("CodeAt", ctx, addr, blockNumber).Return(code, nil).Once()
			mockClient.On("CallContract", ctx, mock.Anything, mock.Anything).Return(code, nil).Times(3)
			mockStore.On("InsertERC20", mock.Anything).Return(nil).Once()
			res, err := srv.AddERC20(ctx, req)
			Expect(err).Should(BeNil())
			Expect(res).ShouldNot(BeNil())
		})
		It("success even if failed to call contract", func() {
			addr := ethCommon.HexToAddress("0x01")
			blockNumber := big.NewInt(100)
			code := []byte("1234567890")
			req := &pb.AddERC20Request{
				Address:     addr.Hex(),
				BlockNumber: blockNumber.Int64(),
			}
			mockClient.On("CodeAt", ctx, addr, blockNumber).Return(code, nil).Once()
			mockClient.On("CallContract", ctx, mock.Anything, mock.Anything).Return(nil, unknownErr).Times(3)
			mockStore.On("InsertERC20", mock.Anything).Return(nil).Once()
			res, err := srv.AddERC20(ctx, req)
			Expect(err).Should(BeNil())
			Expect(res).ShouldNot(BeNil())
		})
		Context("with invalid parameters", func() {
			It("failed to insert to db", func() {
				addr := ethCommon.HexToAddress("0x01")
				blockNumber := big.NewInt(100)
				code := []byte("1234567890")
				req := &pb.AddERC20Request{
					Address:     addr.Hex(),
					BlockNumber: blockNumber.Int64(),
				}
				mockClient.On("CodeAt", ctx, addr, blockNumber).Return(code, nil).Once()
				mockClient.On("CallContract", ctx, mock.Anything, mock.Anything).Return(code, nil).Times(3)
				mockStore.On("InsertERC20", mock.Anything).Return(unknownErr).Once()
				res, err := srv.AddERC20(ctx, req)
				Expect(err).Should(Equal(ErrInternal))
				Expect(res).Should(BeNil())
			})
			It("failed to get code due to unknown error", func() {
				addr := ethCommon.HexToAddress("0x01")
				blockNumber := big.NewInt(100)
				req := &pb.AddERC20Request{
					Address:     addr.Hex(),
					BlockNumber: blockNumber.Int64(),
				}
				mockClient.On("CodeAt", ctx, addr, blockNumber).Return(nil, unknownErr).Once()
				res, err := srv.AddERC20(ctx, req)
				Expect(err).Should(Equal(ErrInternal))
				Expect(res).Should(BeNil())
			})
			It("failed to get code due to ethereum not found error", func() {
				addr := ethCommon.HexToAddress("0x01")
				blockNumber := big.NewInt(100)
				req := &pb.AddERC20Request{
					Address:     addr.Hex(),
					BlockNumber: blockNumber.Int64(),
				}
				mockClient.On("CodeAt", ctx, addr, blockNumber).Return(nil, ethereum.NotFound).Once()
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
