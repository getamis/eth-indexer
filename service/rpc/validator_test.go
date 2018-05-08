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
package rpc

import (
	"context"

	"github.com/getamis/sirius/log"
	"github.com/maichain/eth-indexer/service/pb"
	"github.com/maichain/eth-indexer/service/rpc/mocks"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Validator Test", func() {
	var (
		mockServer *mocks.Server
		svr        *validatingMiddleware
	)

	BeforeEach(func() {
		mockServer = new(mocks.Server)
		svr = &validatingMiddleware{
			logger: log.Discard(),
			next:   mockServer,
		}
	})

	AfterEach(func() {
		mockServer.AssertExpectations(GinkgoT())
	})

	DescribeTable("GetBlockByHash()",
		func(hash string, ok bool) {
			ctx := context.Background()
			req := &pb.BlockHashQueryRequest{
				Hash: hash,
			}
			expRes := &pb.BlockQueryResponse{}
			if ok {
				mockServer.On("GetBlockByHash", ctx, req).Return(expRes, nil).Once()
			}
			res, err := svr.GetBlockByHash(ctx, req)
			if ok {
				Expect(res).Should(Equal(expRes))
				Expect(err).Should(BeNil())
			} else {
				Expect(res).Should(BeNil())
				Expect(err).Should(Equal(ErrInvalidHash))
			}
		},
		Entry("valid hash", "0x35b9253b70be351059982e8d6a218146a18ef9b723e560c7efc540629b4e75f2", true),
		Entry("valid hash with 0x prefix", "35b9253b70be351059982e8d6a218146a18ef9b723e560c7efc540629b4e75f2", true),
		Entry("invalid hash with invalid characters", "0x35b9253b70be351059982e8d6a218146a18ef9b723e560c7efc540629b4e75fZ", false),
		Entry("invalid hash with invalid length", "0x35b9253b70be351059982e8d6a218146a18ef9b723e560c7efc540629b4e75", false),
	)

	DescribeTable("GetBlockByNumber()",
		func(number int64, ok bool) {
			ctx := context.Background()
			req := &pb.BlockNumberQueryRequest{
				Number: number,
			}
			expRes := &pb.BlockQueryResponse{}
			if ok {
				mockServer.On("GetBlockByNumber", ctx, req).Return(expRes, nil).Once()
			}
			res, err := svr.GetBlockByNumber(ctx, req)
			if ok {
				Expect(res).Should(Equal(expRes))
				Expect(err).Should(BeNil())
			} else {
				Expect(res).Should(BeNil())
				Expect(err).Should(Equal(ErrInvalidBlockNumber))
			}
		},
		Entry("valid number", int64(1000), true),
		Entry("latest block number", int64(-1), true),
		Entry("invalid number", int64(-2), false),
	)

	DescribeTable("GetTransactionByHash()",
		func(hash string, ok bool) {
			ctx := context.Background()
			req := &pb.TransactionQueryRequest{
				Hash: hash,
			}
			expRes := &pb.TransactionQueryResponse{}
			if ok {
				mockServer.On("GetTransactionByHash", ctx, req).Return(expRes, nil).Once()
			}
			res, err := svr.GetTransactionByHash(ctx, req)
			if ok {
				Expect(res).Should(Equal(expRes))
				Expect(err).Should(BeNil())
			} else {
				Expect(res).Should(BeNil())
				Expect(err).Should(Equal(ErrInvalidHash))
			}
		},
		Entry("valid hash", "0x35b9253b70be351059982e8d6a218146a18ef9b723e560c7efc540629b4e75f2", true),
		Entry("valid hash with 0x prefix", "35b9253b70be351059982e8d6a218146a18ef9b723e560c7efc540629b4e75f2", true),
		Entry("invalid hash with invalid characters", "0x35b9253b70be351059982e8d6a218146a18ef9b723e560c7efc540629b4e75fZ", false),
		Entry("invalid hash with invalid length", "0x35b9253b70be351059982e8d6a218146a18ef9b723e560c7efc540629b4e75", false),
	)

	DescribeTable("GetBalance()",
		func(number int64, address string, token string, err error) {
			ctx := context.Background()
			req := &pb.GetBalanceRequest{
				BlockNumber: number,
				Address:     address,
				Token:       token,
			}
			expRes := &pb.GetBalanceResponse{}
			if err == nil {
				mockServer.On("GetBalance", ctx, req).Return(expRes, nil).Once()
			}
			res, err := svr.GetBalance(ctx, req)
			if err == nil {
				Expect(res).Should(Equal(expRes))
				Expect(err).Should(BeNil())
			} else {
				Expect(res).Should(BeNil())
				Expect(err).Should(Equal(err))
			}
		},
		Entry("valid parameters for eth token", int64(-1), "0x343c43a37d37dff08ae8c4a11544c718abb4fcf8", ethToken, nil),
		Entry("valid parameters for a erc20 token", int64(-1), "0x343c43a37d37dff08ae8c4a11544c718abb4fcf8", "0x343c43a37d37dff08ae8c4a11544c718abb4fcf8", nil),
		Entry("invalid parameters with invalid block number", int64(-2), "0x343c43a37d37dff08ae8c4a11544c718abb4fcf8", ethToken, ErrInvalidBlockNumber),
		Entry("invalid parameters with invalid address", int64(-2), "0x343c43a37d37dff08ae8c4a11544c718abb4fc", ethToken, ErrInvalidAddress),
		Entry("invalid parameters with invalid token address", int64(-2), "0x343c43a37d37dff08ae8c4a11544c718abb4fc", "wrong token", ErrInvalidToken),
	)

	DescribeTable("GetOffsetBalance()",
		func(offset int64, address string, token string, err error) {
			ctx := context.Background()
			req := &pb.GetOffsetBalanceRequest{
				Offset:  offset,
				Address: address,
				Token:   token,
			}
			expRes := &pb.GetBalanceResponse{}
			if err == nil {
				mockServer.On("GetOffsetBalance", ctx, req).Return(expRes, nil).Once()
			}
			res, err := svr.GetOffsetBalance(ctx, req)
			if err == nil {
				Expect(res).Should(Equal(expRes))
				Expect(err).Should(BeNil())
			} else {
				Expect(res).Should(BeNil())
				Expect(err).Should(Equal(err))
			}
		},
		Entry("valid parameters for eth token", int64(10), "0x343c43a37d37dff08ae8c4a11544c718abb4fcf8", ethToken, nil),
		Entry("valid parameters for a erc20 token", int64(10), "0x343c43a37d37dff08ae8c4a11544c718abb4fcf8", "0x343c43a37d37dff08ae8c4a11544c718abb4fcf8", nil),
		Entry("invalid parameters with invalid offset", int64(-1), "0x343c43a37d37dff08ae8c4a11544c718abb4fcf8", ethToken, ErrInvalidBlockNumber),
		Entry("invalid parameters with invalid address", int64(10), "0x343c43a37d37dff08ae8c4a11544c718abb4fc", ethToken, ErrInvalidAddress),
		Entry("invalid parameters with invalid token address", int64(10), "0x343c43a37d37dff08ae8c4a11544c718abb4fc", "wrong token", ErrInvalidToken),
	)
})
