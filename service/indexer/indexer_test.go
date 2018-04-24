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
package indexer

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/maichain/eth-indexer/model"
	indexerMocks "github.com/maichain/eth-indexer/service/indexer/mocks"
	storeMocks "github.com/maichain/eth-indexer/store/mocks"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

var _ = Describe("Indexer Test", func() {
	var (
		mockEthClient    *indexerMocks.EthClient
		mockStoreManager *storeMocks.Manager
		idx              *indexer
	)
	BeforeSuite(func() {
		mockStoreManager = new(storeMocks.Manager)
		mockEthClient = new(indexerMocks.EthClient)
		idx = New(mockEthClient, mockStoreManager)
	})

	AfterSuite(func() {
		mockStoreManager.AssertExpectations(GinkgoT())
		mockEthClient.AssertExpectations(GinkgoT())
	})

	Context("Listen()", func() {
		ctx, cancel := context.WithCancel(context.Background())
		ch := make(chan *types.Header)
		unknownErr := errors.New("unknown error")

		It("should be ok", func() {
			// blocks from 11 to 15 are ethereum
			// receive 18, 19 blocks from header channel
			mockStoreManager.On("LatestHeader").Return(&model.Header{
				Number: 10,
			}, nil).Once()
			mockStoreManager.On("LatestStateBlock").Return(&model.StateBlock{
				Number: 10,
			}, nil).Once()

			var num *big.Int
			mockEthClient.On("BlockByNumber", mock.Anything, num).Return(types.NewBlockWithHeader(
				&types.Header{
					Number: big.NewInt(15),
				},
			), nil).Once()

			tx := types.NewTransaction(0, common.Address{}, common.Big0, 0, common.Big0, []byte{})
			receipt := types.NewReceipt([]byte{}, false, 0)
			for i := int64(11); i <= 19; i++ {
				block := types.NewBlock(
					&types.Header{
						Number: big.NewInt(i),
						Root:   common.StringToHash("1234567890" + strconv.Itoa(int(i))),
					}, []*types.Transaction{tx}, nil, []*types.Receipt{receipt})
				mockEthClient.On("BlockByNumber", mock.Anything, big.NewInt(i)).Return(block, nil).Once()
				mockStoreManager.On("InsertBlock", block, []*types.Receipt{receipt}).Return(nil).Once()

				// Sometimes we cannot get account states successfully
				if i%2 == 0 {
					dump := &state.Dump{
						Root: fmt.Sprintf("%x", block.Root()),
					}
					mockEthClient.On("ModifiedAccountStatesByNumber", mock.Anything, uint64(i-2), uint64(i)).Return(dump, nil).Once()
					mockStoreManager.On("UpdateState", block, dump).Return(nil).Once()
				} else {
					mockEthClient.On("ModifiedAccountStatesByNumber", mock.Anything, uint64(i-1), uint64(i)).Return(nil, unknownErr).Once()
				}
			}
			mockEthClient.On("TransactionReceipt", mock.Anything, tx.Hash()).Return(receipt, nil).Times(9)
			var recvCh chan<- *types.Header
			recvCh = ch
			mockEthClient.On("SubscribeNewHead", mock.Anything, recvCh).Return(nil, nil).Once()

			go func() {
				ch <- &types.Header{
					Number: big.NewInt(18),
				}
				ch <- &types.Header{
					Number: big.NewInt(19),
				}
				time.Sleep(time.Second)
				cancel()
			}()

			err := idx.Listen(ctx, ch)
			Expect(err).Should(Equal(context.Canceled))
		})

		Context("with something wrong", func() {
			It("failed to subscribe new head", func() {
				mockStoreManager.On("LatestHeader").Return(&model.Header{
					Number: 10,
				}, nil).Once()
				mockStoreManager.On("LatestStateBlock").Return(&model.StateBlock{
					Number: 10,
				}, nil).Once()

				var num *big.Int
				mockEthClient.On("BlockByNumber", mock.Anything, num).Return(types.NewBlockWithHeader(
					&types.Header{
						Number: big.NewInt(15),
					},
				), nil).Once()

				tx := types.NewTransaction(0, common.Address{}, common.Big0, 0, common.Big0, []byte{})
				receipt := types.NewReceipt([]byte{}, false, 0)
				for i := int64(11); i <= 15; i++ {
					block := types.NewBlock(
						&types.Header{
							Number: big.NewInt(i),
							Root:   common.StringToHash("1234567890" + strconv.Itoa(int(i))),
						}, []*types.Transaction{tx}, nil, []*types.Receipt{receipt})
					mockEthClient.On("BlockByNumber", mock.Anything, big.NewInt(i)).Return(block, nil).Once()
					mockStoreManager.On("InsertBlock", block, []*types.Receipt{receipt}).Return(nil).Once()

					// Sometimes we cannot get account states successfully
					if i%2 == 0 {
						dump := &state.Dump{
							Root: fmt.Sprintf("%x", block.Root()),
						}
						mockEthClient.On("ModifiedAccountStatesByNumber", mock.Anything, uint64(i-2), uint64(i)).Return(dump, nil).Once()
						mockStoreManager.On("UpdateState", block, dump).Return(nil).Once()
					} else {
						mockEthClient.On("ModifiedAccountStatesByNumber", mock.Anything, uint64(i-1), uint64(i)).Return(nil, unknownErr).Once()
					}
				}
				mockEthClient.On("TransactionReceipt", mock.Anything, tx.Hash()).Return(receipt, nil).Times(5)
				var recvCh chan<- *types.Header
				recvCh = ch
				mockEthClient.On("SubscribeNewHead", mock.Anything, recvCh).Return(nil, unknownErr).Once()

				err := idx.Listen(ctx, ch)
				Expect(err).Should(Equal(unknownErr))
			})

			It("failed to insert block to db", func() {
				mockStoreManager.On("LatestHeader").Return(&model.Header{
					Number: 10,
				}, nil).Once()
				mockStoreManager.On("LatestStateBlock").Return(&model.StateBlock{
					Number: 10,
				}, nil).Once()
				var num *big.Int
				mockEthClient.On("BlockByNumber", mock.Anything, num).Return(types.NewBlockWithHeader(
					&types.Header{
						Number: big.NewInt(15),
					},
				), nil).Once()

				tx := types.NewTransaction(0, common.Address{}, common.Big0, 0, common.Big0, []byte{})
				receipt := types.NewReceipt([]byte{}, false, 0)
				block := types.NewBlock(
					&types.Header{
						Number: big.NewInt(11),
					}, []*types.Transaction{tx}, nil, []*types.Receipt{receipt})
				mockEthClient.On("BlockByNumber", mock.Anything, big.NewInt(11)).Return(block, nil).Once()
				mockEthClient.On("TransactionReceipt", mock.Anything, tx.Hash()).Return(receipt, nil).Once()
				mockStoreManager.On("InsertBlock", block, []*types.Receipt{receipt}).Return(unknownErr).Once()

				err := idx.Listen(ctx, ch)
				Expect(err).Should(Equal(unknownErr))
			})

			It("failed to get transaction receipt", func() {
				mockStoreManager.On("LatestHeader").Return(&model.Header{
					Number: 10,
				}, nil).Once()
				mockStoreManager.On("LatestStateBlock").Return(&model.StateBlock{
					Number: 10,
				}, nil).Once()
				var num *big.Int
				mockEthClient.On("BlockByNumber", mock.Anything, num).Return(types.NewBlockWithHeader(
					&types.Header{
						Number: big.NewInt(15),
					},
				), nil).Once()

				tx := types.NewTransaction(0, common.Address{}, common.Big0, 0, common.Big0, []byte{})
				receipt := types.NewReceipt([]byte{}, false, 0)
				block := types.NewBlock(
					&types.Header{
						Number: big.NewInt(11),
					}, []*types.Transaction{tx}, nil, []*types.Receipt{receipt})
				mockEthClient.On("BlockByNumber", mock.Anything, big.NewInt(11)).Return(block, nil).Once()
				mockEthClient.On("TransactionReceipt", mock.Anything, tx.Hash()).Return(nil, unknownErr).Once()

				err := idx.Listen(ctx, ch)
				Expect(err).Should(Equal(unknownErr))
			})

			It("failed to get block by number", func() {
				mockStoreManager.On("LatestHeader").Return(&model.Header{
					Number: 10,
				}, nil).Once()
				mockStoreManager.On("LatestStateBlock").Return(&model.StateBlock{
					Number: 10,
				}, nil).Once()
				var num *big.Int
				mockEthClient.On("BlockByNumber", mock.Anything, num).Return(types.NewBlockWithHeader(
					&types.Header{
						Number: big.NewInt(15),
					},
				), nil).Once()
				mockEthClient.On("BlockByNumber", mock.Anything, big.NewInt(11)).Return(nil, unknownErr).Once()

				err := idx.Listen(ctx, ch)
				Expect(err).Should(Equal(unknownErr))
			})

			It("failed to get block by number (the first time)", func() {
				mockStoreManager.On("LatestHeader").Return(&model.Header{
					Number: 10,
				}, nil).Once()
				mockStoreManager.On("LatestStateBlock").Return(&model.StateBlock{
					Number: 10,
				}, nil).Once()
				var num *big.Int
				mockEthClient.On("BlockByNumber", mock.Anything, num).Return(nil, unknownErr).Once()
				err := idx.Listen(ctx, ch)
				Expect(err).Should(Equal(unknownErr))
			})

			It("failed to get state block", func() {
				mockStoreManager.On("LatestHeader").Return(&model.Header{
					Number: 10,
				}, nil).Once()
				mockStoreManager.On("LatestStateBlock").Return(nil, unknownErr).Once()
				err := idx.Listen(ctx, ch)
				Expect(err).Should(Equal(unknownErr))
			})

			It("failed to get latest header", func() {
				mockStoreManager.On("LatestHeader").Return(nil, unknownErr).Once()
				err := idx.Listen(ctx, ch)
				Expect(err).Should(Equal(unknownErr))
			})
		})
	})
})

func TestBlockHeader(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Indexer Test")
}
