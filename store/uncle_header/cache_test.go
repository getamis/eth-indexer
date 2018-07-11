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

package uncle_header

import (
	"errors"

	"github.com/getamis/eth-indexer/common"
	"github.com/getamis/eth-indexer/model"
	"github.com/getamis/eth-indexer/store/uncle_header/mocks"
	lru "github.com/hashicorp/golang-lru"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Cache Test", func() {
	var (
		mockStore  *mocks.Store
		cacheStore Store
	)
	header := &model.UncleHeader{
		Number: 100,
		Hash:   []byte("456"),
	}
	unknownErr := errors.New("unknown error")
	BeforeEach(func() {
		// Init cache before each tests
		uncleHashCache, _ = lru.NewARC(cacheSize)

		mockStore = new(mocks.Store)
		cacheStore = newCacheMiddleware(mockStore)
	})

	AfterEach(func() {
		mockStore.AssertExpectations(GinkgoT())
	})

	Context("Insert()", func() {
		Context("add in cache", func() {
			It("insert store successfully", func() {
				mockStore.On("Insert", header).Return(nil).Once()
				err := cacheStore.Insert(header)
				Expect(err).Should(BeNil())
				v1, ok := uncleHashCache.Get(common.BytesToHex(header.Hash))
				h1 := v1.(*model.UncleHeader)
				Expect(ok).Should(BeTrue())
				Expect(h1).Should(Equal(header))
			})
			It("failed to insert store due to duplicate key error", func() {
				mockStore.On("Insert", header).Return(duplicateErr).Once()
				err := cacheStore.Insert(header)
				Expect(err).Should(Equal(duplicateErr))
				v1, ok := uncleHashCache.Get(common.BytesToHex(header.Hash))
				h1 := v1.(*model.UncleHeader)
				Expect(ok).Should(BeTrue())
				Expect(h1).Should(Equal(header))
			})
		})
		It("not add in cache", func() {
			mockStore.On("Insert", header).Return(unknownErr).Once()
			err := cacheStore.Insert(header)
			Expect(err).Should(Equal(unknownErr))
			_, ok := uncleHashCache.Get(common.BytesToHex(header.Hash))
			Expect(ok).Should(BeFalse())
		})
	})

	Context("FindUncleByHash()", func() {
		It("in cache", func() {
			By("wrong in cache")
			mockStore.On("FindUncleByHash", header.Hash).Return(nil, unknownErr).Once()
			expHeader, err := cacheStore.FindUncleByHash(header.Hash)
			Expect(err).Should(Equal(unknownErr))
			Expect(expHeader).Should(BeNil())

			By("add in cache")
			mockStore.On("Insert", header).Return(nil).Once()
			err = cacheStore.Insert(header)
			Expect(err).Should(BeNil())

			expHeader, err = cacheStore.FindUncleByHash(header.Hash)
			Expect(err).Should(BeNil())
			Expect(expHeader).Should(Equal(header))
		})
	})
})
