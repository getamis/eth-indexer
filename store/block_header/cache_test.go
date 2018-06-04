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
package block_header

import (
	"errors"

	lru "github.com/hashicorp/golang-lru"
	"github.com/getamis/eth-indexer/common"
	"github.com/getamis/eth-indexer/model"
	"github.com/getamis/eth-indexer/store/block_header/mocks"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Cache Test", func() {
	var (
		mockStore  *mocks.Store
		cacheStore Store
	)
	td := &model.TotalDifficulty{
		Block: 100,
		Hash:  []byte("123"),
		Td:    "10000000",
	}
	header := &model.Header{
		Number: 100,
		Hash:   []byte("456"),
	}
	unknownErr := errors.New("unknown error")
	BeforeEach(func() {
		// Init cache before each tests
		tdCache, _ = lru.NewARC(cacheSize)
		blockHashCache, _ = lru.NewARC(cacheSize)

		mockStore = new(mocks.Store)
		cacheStore = newCacheMiddleware(mockStore)
	})

	AfterEach(func() {
		mockStore.AssertExpectations(GinkgoT())
	})

	Context("InsertTd()", func() {
		It("in cache", func() {
			By("add in cache")
			mockStore.On("InsertTd", td).Return(nil).Once()
			err := cacheStore.InsertTd(td)
			Expect(err).Should(BeNil())

			By("call again, should be duplicate key error")
			err = cacheStore.InsertTd(td)
			Expect(err).Should(Equal(duplicateErr))
		})
		Context("not in cache", func() {
			It("insert store successfully", func() {
				mockStore.On("InsertTd", td).Return(nil).Once()
				err := cacheStore.InsertTd(td)
				Expect(err).Should(BeNil())
				value, ok := tdCache.Get(common.BytesToHex(td.Hash))
				Expect(ok).Should(BeTrue())
				resTD := value.(*model.TotalDifficulty)
				Expect(resTD).Should(Equal(td))
			})
			It("failed to insert store due to duplicate key error", func() {
				mockStore.On("InsertTd", td).Return(duplicateErr).Once()
				err := cacheStore.InsertTd(td)
				Expect(err).Should(Equal(duplicateErr))
				value, ok := tdCache.Get(common.BytesToHex(td.Hash))
				Expect(ok).Should(BeTrue())
				resTD := value.(*model.TotalDifficulty)
				Expect(resTD).Should(Equal(td))
			})

			It("not add in cache", func() {
				mockStore.On("InsertTd", td).Return(unknownErr).Once()
				err := cacheStore.InsertTd(td)
				Expect(err).Should(Equal(unknownErr))
				_, ok := tdCache.Get(common.BytesToHex(td.Hash))
				Expect(ok).Should(BeFalse())
			})
		})
	})

	Context("Insert()", func() {
		Context("add in cache", func() {
			It("insert store successfully", func() {
				mockStore.On("Insert", header).Return(nil).Once()
				err := cacheStore.Insert(header)
				Expect(err).Should(BeNil())
				v1, ok := blockHashCache.Get(common.BytesToHex(header.Hash))
				h1 := v1.(*model.Header)
				Expect(ok).Should(BeTrue())
				Expect(h1).Should(Equal(header))
			})
			It("failed to insert store due to duplicate key error", func() {
				mockStore.On("Insert", header).Return(duplicateErr).Once()
				err := cacheStore.Insert(header)
				Expect(err).Should(Equal(duplicateErr))
				v1, ok := blockHashCache.Get(common.BytesToHex(header.Hash))
				h1 := v1.(*model.Header)
				Expect(ok).Should(BeTrue())
				Expect(h1).Should(Equal(header))
			})
		})
		It("not add in cache", func() {
			mockStore.On("Insert", header).Return(unknownErr).Once()
			err := cacheStore.Insert(header)
			Expect(err).Should(Equal(unknownErr))
			_, ok := blockHashCache.Get(common.BytesToHex(header.Hash))
			Expect(ok).Should(BeFalse())
		})
	})

	Context("FindTd()", func() {
		It("in cache", func() {
			By("wrong in cache")
			tdCache.Add(common.BytesToHex(td.Hash), "wrong data")
			mockStore.On("FindTd", td.Hash).Return(nil, unknownErr).Once()
			expTD, err := cacheStore.FindTd(td.Hash)
			Expect(err).Should(Equal(unknownErr))
			Expect(expTD).Should(BeNil())

			By("add in cache")
			mockStore.On("InsertTd", td).Return(nil).Once()
			err = cacheStore.InsertTd(td)
			Expect(err).Should(BeNil())

			expTD, err = cacheStore.FindTd(td.Hash)
			Expect(err).Should(BeNil())
			Expect(expTD).Should(Equal(td))
		})
		Context("not in cache", func() {
			It("find TD successfully", func() {
				mockStore.On("FindTd", td.Hash).Return(td, nil).Once()
				expTD, err := cacheStore.FindTd(td.Hash)
				Expect(err).Should(BeNil())
				Expect(expTD).Should(Equal(td))

				value, ok := tdCache.Get(common.BytesToHex(td.Hash))
				Expect(ok).Should(BeTrue())
				resTD := value.(*model.TotalDifficulty)
				Expect(resTD).Should(Equal(td))
			})
			It("failed to find TD", func() {
				mockStore.On("FindTd", td.Hash).Return(nil, unknownErr).Once()
				expTD, err := cacheStore.FindTd(td.Hash)
				Expect(err).Should(Equal(unknownErr))
				Expect(expTD).Should(BeNil())

				_, ok := tdCache.Get(common.BytesToHex(td.Hash))
				Expect(ok).Should(BeFalse())
			})
		})
	})

	Context("FindBlockByNumber()", func() {
		It("returns the same response from mockStore", func() {
			number := int64(100)
			mockStore.On("FindBlockByNumber", number).Return(nil, unknownErr).Once()
			expHeader, err := cacheStore.FindBlockByNumber(number)
			Expect(err).Should(Equal(unknownErr))
			Expect(expHeader).Should(BeNil())
		})
	})

	Context("FindBlockByHash()", func() {
		It("in cache", func() {
			By("wrong in cache")
			tdCache.Add(common.BytesToHex(header.Hash), "wrong data")
			mockStore.On("FindBlockByHash", header.Hash).Return(nil, unknownErr).Once()
			expHeader, err := cacheStore.FindBlockByHash(header.Hash)
			Expect(err).Should(Equal(unknownErr))
			Expect(expHeader).Should(BeNil())

			By("add in cache")
			mockStore.On("Insert", header).Return(nil).Once()
			err = cacheStore.Insert(header)
			Expect(err).Should(BeNil())

			expHeader, err = cacheStore.FindBlockByHash(header.Hash)
			Expect(err).Should(BeNil())
			Expect(expHeader).Should(Equal(header))
		})
		Context("not in cache", func() {
			It("find TD successfully", func() {
				mockStore.On("FindBlockByHash", header.Hash).Return(header, nil).Once()
				expHeader, err := cacheStore.FindBlockByHash(header.Hash)
				Expect(err).Should(BeNil())
				Expect(expHeader).Should(Equal(header))

				value, ok := blockHashCache.Get(common.BytesToHex(header.Hash))
				Expect(ok).Should(BeTrue())
				resHeader := value.(*model.Header)
				Expect(resHeader).Should(Equal(header))
			})
			It("failed to find TD", func() {
				mockStore.On("FindBlockByHash", header.Hash).Return(nil, unknownErr).Once()
				expHeader, err := cacheStore.FindBlockByHash(header.Hash)
				Expect(err).Should(Equal(unknownErr))
				Expect(expHeader).Should(BeNil())

				_, ok := blockHashCache.Get(common.BytesToHex(header.Hash))
				Expect(ok).Should(BeFalse())
			})
		})
	})

	Context("FindBlockByNumber()", func() {
		It("find block by number successfully", func() {
			mockStore.On("FindBlockByNumber", header.Number).Return(header, nil).Once()
			expHeader, err := cacheStore.FindBlockByNumber(header.Number)
			Expect(err).Should(BeNil())
			Expect(expHeader).Should(Equal(header))

			v1, ok := blockHashCache.Get(common.BytesToHex(header.Hash))
			h1 := v1.(*model.Header)
			Expect(ok).Should(BeTrue())
			Expect(h1).Should(Equal(header))
		})

		It("failed to find block by number", func() {
			mockStore.On("FindBlockByNumber", header.Number).Return(nil, unknownErr).Once()
			expHeader, err := cacheStore.FindBlockByNumber(header.Number)
			Expect(err).Should(Equal(unknownErr))
			Expect(expHeader).Should(BeNil())

			_, ok := blockHashCache.Get(common.BytesToHex(header.Hash))
			Expect(ok).Should(BeFalse())
		})
	})

	Context("FindLatestBlock()", func() {
		It("find latest block successfully", func() {
			mockStore.On("FindLatestBlock").Return(header, nil).Once()
			expHeader, err := cacheStore.FindLatestBlock()
			Expect(err).Should(BeNil())
			Expect(expHeader).Should(Equal(header))

			v1, ok := blockHashCache.Get(common.BytesToHex(header.Hash))
			h1 := v1.(*model.Header)
			Expect(ok).Should(BeTrue())
			Expect(h1).Should(Equal(header))
		})

		It("failed to find latest block", func() {
			mockStore.On("FindLatestBlock").Return(nil, unknownErr).Once()
			expHeader, err := cacheStore.FindLatestBlock()
			Expect(err).Should(Equal(unknownErr))
			Expect(expHeader).Should(BeNil())

			_, ok := blockHashCache.Get(common.BytesToHex(header.Hash))
			Expect(ok).Should(BeFalse())
		})
	})
})
