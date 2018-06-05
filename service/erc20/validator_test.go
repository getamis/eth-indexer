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

package erc20

import (
	"context"

	"github.com/getamis/eth-indexer/service"
	"github.com/getamis/eth-indexer/service/erc20/mocks"
	"github.com/getamis/eth-indexer/service/pb"
	"github.com/getamis/sirius/log"
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
