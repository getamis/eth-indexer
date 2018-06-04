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

	"github.com/getamis/sirius/log"
	"github.com/getamis/eth-indexer/service"
	"github.com/getamis/eth-indexer/service/erc20/mocks"
	"github.com/getamis/eth-indexer/service/pb"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Validator Test", func() {
	var (
		mockServer *mocks.ERC20ServiceServer
		srv        *validatingMiddleware
	)
	BeforeEach(func() {
		mockServer = new(mocks.ERC20ServiceServer)
		srv = &validatingMiddleware{
			logger: log.New(),
			next:   mockServer,
		}
	})
	AfterEach(func() {
		mockServer.AssertExpectations(GinkgoT())
	})

	Context("AddERC20()", func() {
		ctx := context.Background()
		It("with valid parameters", func() {
			req := &pb.AddERC20Request{
				Address:     "0x1234567890123456789012345678901234567890",
				BlockNumber: 100,
			}
			expRes := &pb.AddERC20Response{
				Address:     "0x1234567890123456789012345678901234567890",
				BlockNumber: 100,
				TotalSupply: "100",
				Decimals:    18,
				Name:        "name",
			}
			mockServer.On("AddERC20", ctx, req).Return(expRes, nil).Once()
			res, err := srv.AddERC20(ctx, req)
			Expect(err).Should(BeNil())
			Expect(res).Should(Equal(expRes))
		})

		Context("with invalid parameters", func() {
			It("invalid address", func() {
				req := &pb.AddERC20Request{
					Address:     "0x123456789012345678901234567890123456789Z",
					BlockNumber: 100,
				}
				res, err := srv.AddERC20(ctx, req)
				Expect(err).Should(Equal(service.ErrInvalidAddress))
				Expect(res).Should(BeNil())
			})

			It("invalid block number", func() {
				req := &pb.AddERC20Request{
					Address:     "0x1234567890123456789012345678901234567890",
					BlockNumber: -1,
				}
				res, err := srv.AddERC20(ctx, req)
				Expect(err).Should(Equal(service.ErrInvalidBlockNumber))
				Expect(res).Should(BeNil())
			})
		})
	})
})
