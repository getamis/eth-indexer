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

package client

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/getamis/eth-indexer/client/mocks"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Cache Test", func() {
	var (
		mockClient  *mocks.EthClient
		cacheClient *cacheMiddleware
	)

	ctx := context.Background()
	td := big.NewInt(100)
	block := types.NewBlockWithHeader(
		&types.Header{
			Number:     big.NewInt(100),
			Root:       common.HexToHash("12345678900"),
			Difficulty: td,
		},
	)
	tx := types.NewTransaction(1, common.HexToAddress("1234567890"), big.NewInt(1000), 100, big.NewInt(10000), nil)
	unknownErr := errors.New("unknown error")

	BeforeEach(func() {
		mockClient = new(mocks.EthClient)
		cacheClient = newCacheMiddleware(mockClient).(*cacheMiddleware)
	})

	AfterEach(func() {
		mockClient.AssertExpectations(GinkgoT())

		txCache.Purge()
		tdCache.Purge()
		blockCache.Purge()
		blockReceiptsCache.Purge()
	})

	Context("BlockByHash()", func() {
		It("in cache", func() {
			By("wrong in cache")
			blockCache.Add(block.Hash().Hex(), "wrong data")
			mockClient.On("BlockByHash", ctx, block.Hash()).Return(nil, unknownErr).Once()
			resBlock, err := cacheClient.BlockByHash(ctx, block.Hash())
			Expect(err).Should(Equal(unknownErr))
			Expect(resBlock).Should(BeNil())

			By("add in cache")
			mockClient.On("BlockByHash", ctx, block.Hash()).Return(block, nil).Once()
			resBlock, err = cacheClient.BlockByHash(ctx, block.Hash())
			Expect(err).Should(BeNil())
			Expect(resBlock).Should(Equal(block))

			By("already in cache")
			resBlock, err = cacheClient.BlockByHash(ctx, block.Hash())
			Expect(err).Should(BeNil())
			Expect(resBlock).Should(Equal(block))
		})
		Context("not in cache", func() {
			It("find block successfully", func() {
				mockClient.On("BlockByHash", ctx, block.Hash()).Return(block, nil).Once()
				resBlock, err := cacheClient.BlockByHash(ctx, block.Hash())
				Expect(err).Should(BeNil())
				Expect(resBlock).Should(Equal(block))

				resBlock, err = cacheClient.BlockByHash(ctx, block.Hash())
				Expect(err).Should(BeNil())
				Expect(resBlock).Should(Equal(block))
			})
			It("failed to find block", func() {
				mockClient.On("BlockByHash", ctx, block.Hash()).Return(nil, unknownErr).Once()
				resBlock, err := cacheClient.BlockByHash(ctx, block.Hash())
				Expect(err).Should(Equal(unknownErr))
				Expect(resBlock).Should(BeNil())

				_, ok := blockCache.Get(block.Hash().Hex())
				Expect(ok).Should(BeFalse())
			})
		})
	})

	Context("TransactionByHash()", func() {
		It("in cache", func() {
			By("wrong in cache")
			txCache.Add(tx.Hash().Hex(), "wrong data")
			mockClient.On("TransactionByHash", ctx, tx.Hash()).Return(nil, false, unknownErr).Once()
			resTX, _, err := cacheClient.TransactionByHash(ctx, tx.Hash())
			Expect(err).Should(Equal(unknownErr))
			Expect(resTX).Should(BeNil())

			By("add in cache")
			mockClient.On("TransactionByHash", ctx, tx.Hash()).Return(tx, true, nil).Once()
			resTX, pending, err := cacheClient.TransactionByHash(ctx, tx.Hash())
			Expect(err).Should(BeNil())
			Expect(pending).Should(BeTrue())
			Expect(resTX).Should(Equal(tx))

			By("already in cache")
			resTX, pending, err = cacheClient.TransactionByHash(ctx, tx.Hash())
			Expect(err).Should(BeNil())
			Expect(pending).Should(BeFalse())
			Expect(resTX).Should(Equal(tx))
		})
		Context("not in cache", func() {
			It("find tx successfully", func() {
				mockClient.On("TransactionByHash", ctx, tx.Hash()).Return(tx, true, nil).Once()
				resTX, pending, err := cacheClient.TransactionByHash(ctx, tx.Hash())
				Expect(err).Should(BeNil())
				Expect(pending).Should(BeTrue())
				Expect(resTX).Should(Equal(tx))

				resTX, pending, err = cacheClient.TransactionByHash(ctx, tx.Hash())
				Expect(err).Should(BeNil())
				Expect(pending).Should(BeFalse())
				Expect(resTX).Should(Equal(tx))
			})
			It("failed to find tx", func() {
				mockClient.On("TransactionByHash", ctx, tx.Hash()).Return(nil, false, unknownErr).Once()
				resTX, pending, err := cacheClient.TransactionByHash(ctx, tx.Hash())
				Expect(err).Should(Equal(unknownErr))
				Expect(pending).Should(BeFalse())
				Expect(resTX).Should(BeNil())

				_, ok := txCache.Get(tx.Hash().Hex())
				Expect(ok).Should(BeFalse())
			})
		})
	})

	Context("GetTotalDifficulty()", func() {
		It("in cache", func() {
			By("wrong in cache")
			tdCache.Add(block.Hash().Hex(), "wrong data")
			mockClient.On("GetTotalDifficulty", ctx, block.Hash()).Return(nil, unknownErr).Once()
			resTD, err := cacheClient.GetTotalDifficulty(ctx, block.Hash())
			Expect(err).Should(Equal(unknownErr))
			Expect(resTD).Should(BeNil())

			By("add in cache")
			mockClient.On("GetTotalDifficulty", ctx, block.Hash()).Return(td, nil).Once()
			resTD, err = cacheClient.GetTotalDifficulty(ctx, block.Hash())
			Expect(err).Should(BeNil())
			Expect(resTD).Should(Equal(td))

			By("already in cache")
			resTD, err = cacheClient.GetTotalDifficulty(ctx, block.Hash())
			Expect(err).Should(BeNil())
			Expect(resTD).Should(Equal(td))
		})
		Context("not in cache", func() {
			It("find block successfully", func() {
				mockClient.On("GetTotalDifficulty", ctx, block.Hash()).Return(td, nil).Once()
				resTD, err := cacheClient.GetTotalDifficulty(ctx, block.Hash())
				Expect(err).Should(BeNil())
				Expect(resTD).Should(Equal(td))

				resTD, err = cacheClient.GetTotalDifficulty(ctx, block.Hash())
				Expect(err).Should(BeNil())
				Expect(resTD).Should(Equal(td))
			})
			It("failed to find block", func() {
				mockClient.On("GetTotalDifficulty", ctx, block.Hash()).Return(nil, unknownErr).Once()
				resTD, err := cacheClient.GetTotalDifficulty(ctx, block.Hash())
				Expect(err).Should(Equal(unknownErr))
				Expect(resTD).Should(BeNil())

				_, ok := tdCache.Get(block.Hash().Hex())
				Expect(ok).Should(BeFalse())
			})
		})
	})

	Context("GetBlockReceipts()", func() {
		receipts := types.Receipts{types.NewReceipt([]byte{}, false, 0)}
		It("in cache", func() {
			By("wrong in cache")
			blockReceiptsCache.Add(block.Hash().Hex(), "wrong data")
			mockClient.On("GetBlockReceipts", ctx, block.Hash()).Return(nil, unknownErr).Once()
			got, err := cacheClient.GetBlockReceipts(ctx, block.Hash())
			Expect(err).Should(Equal(unknownErr))
			Expect(got).Should(BeNil())

			By("add in cache")
			mockClient.On("GetBlockReceipts", ctx, block.Hash()).Return(receipts, nil).Once()
			got, err = cacheClient.GetBlockReceipts(ctx, block.Hash())
			Expect(err).Should(BeNil())
			Expect(got).Should(Equal(receipts))

			By("already in cache")
			got, err = cacheClient.GetBlockReceipts(ctx, block.Hash())
			Expect(err).Should(BeNil())
			Expect(got).Should(Equal(receipts))
		})
		Context("not in cache", func() {
			It("find block successfully", func() {
				mockClient.On("GetBlockReceipts", ctx, block.Hash()).Return(receipts, nil).Once()
				got, err := cacheClient.GetBlockReceipts(ctx, block.Hash())
				Expect(err).Should(BeNil())
				Expect(got).Should(Equal(receipts))

				got, err = cacheClient.GetBlockReceipts(ctx, block.Hash())
				Expect(err).Should(BeNil())
				Expect(got).Should(Equal(receipts))
			})
			It("failed to find block", func() {
				mockClient.On("GetBlockReceipts", ctx, block.Hash()).Return(nil, unknownErr).Once()
				got, err := cacheClient.GetBlockReceipts(ctx, block.Hash())
				Expect(err).Should(Equal(unknownErr))
				Expect(got).Should(BeNil())

				_, ok := blockReceiptsCache.Get(block.Hash().Hex())
				Expect(ok).Should(BeFalse())
			})
		})
	})
})

func TestClientServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Client Test")
}
