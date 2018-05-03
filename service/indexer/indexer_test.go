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
	BeforeEach(func() {
		mockStoreManager = new(storeMocks.Manager)
		mockEthClient = new(indexerMocks.EthClient)
		idx = New(mockEthClient, mockStoreManager)
	})

	AfterEach(func() {
		mockStoreManager.AssertExpectations(GinkgoT())
		mockEthClient.AssertExpectations(GinkgoT())
	})

	Context("SyncToTarget()", func() {
		unknownErr := errors.New("unknown error")
		targetBlock := int64(19)

		It("sync to target", func() {
			blocks := make([]*types.Block, 20)
			tx := types.NewTransaction(0, common.Address{}, common.Big0, 0, common.Big0, []byte{})
			receipt := types.NewReceipt([]byte{}, false, 0)
			block := types.NewBlock(
				&types.Header{
					Number: big.NewInt(10),
					Root:   common.StringToHash("1234567890" + strconv.Itoa(int(10))),
				}, []*types.Transaction{tx}, nil, []*types.Receipt{receipt})
			blocks[10] = block
			mockStoreManager.On("LatestHeader").Return(&model.Header{
				Number: 10,
				Hash:   block.Hash().Bytes(),
			}, nil).Once()
			mockStoreManager.On("LatestStateBlock").Return(&model.StateBlock{
				Number: 10,
			}, nil).Once()
			for i := int64(11); i <= targetBlock; i++ {
				block = types.NewBlock(
					&types.Header{
						Number:     big.NewInt(i),
						ParentHash: blocks[i-1].Hash(),
						Root:       common.StringToHash("1234567890" + strconv.Itoa(int(i))),
					}, []*types.Transaction{tx}, nil, []*types.Receipt{receipt})
				blocks[i] = block
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

			err := idx.SyncToTarget(context.Background(), targetBlock)
			Expect(err).Should(BeNil())
		})

		Context("bad target block", func() {
			It("exits right away", func() {
				mockStoreManager.On("LatestHeader").Return(&model.Header{
					Number: 10,
				}, nil).Once()
				mockStoreManager.On("LatestStateBlock").Return(&model.StateBlock{
					Number: 10,
				}, nil).Once()

				err := idx.SyncToTarget(context.Background(), 10)
				Expect(err).ShouldNot(BeNil())
			})
		})
	})

	Context("Listen() without syncing missing blocks", func() {
		ctx, cancel := context.WithCancel(context.Background())
		ch := make(chan *types.Header)
		unknownErr := errors.New("unknown error")

		It("does not sync missing blocks", func() {
			// receive 11-19 blocks from header channel
			tx := types.NewTransaction(0, common.Address{}, common.Big0, 0, common.Big0, []byte{})
			receipt := types.NewReceipt([]byte{}, false, 0)
			blocks := make([]*types.Block, 20)
			parentHash := common.BytesToHash([]byte{})
			for i := int64(11); i <= 19; i++ {
				block := types.NewBlock(
					&types.Header{
						Number:     big.NewInt(i),
						ParentHash: parentHash,
						Root:       common.StringToHash("1234567890" + strconv.Itoa(int(i))),
					}, []*types.Transaction{tx}, nil, []*types.Receipt{receipt})
				blocks[i] = block
				parentHash = block.Hash()
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
				for j := int64(11); j <= 19; j++ {
					ch <- blocks[j].Header()
				}
				time.Sleep(time.Second)
				cancel()
			}()

			err := idx.Listen(ctx, ch, false)
			Expect(err).Should(Equal(context.Canceled))
		})
	})

	Context("Listen()", func() {
		ctx, cancel := context.WithCancel(context.Background())
		ch := make(chan *types.Header)
		unknownErr := errors.New("unknown error")

		It("should be ok", func() {
			// local state has block 10,
			// initial sync blocks 11-15 from ethereum
			// receive 18, 19 blocks from header channel
			blocks := make([]*types.Block, 20)
			tx := types.NewTransaction(0, common.Address{}, common.Big0, 0, common.Big0, []byte{})
			receipt := types.NewReceipt([]byte{}, false, 0)
			block := types.NewBlock(
				&types.Header{
					Number: big.NewInt(10),
					Root:   common.StringToHash("1234567890" + strconv.Itoa(int(10))),
				}, []*types.Transaction{tx}, nil, []*types.Receipt{receipt})
			blocks[10] = block
			mockStoreManager.On("LatestHeader").Return(&model.Header{
				Number: 10,
				Hash:   block.Hash().Bytes(),
			}, nil).Once()
			mockStoreManager.On("LatestStateBlock").Return(&model.StateBlock{
				Number: 10,
			}, nil).Once()

			for i := int64(11); i <= 19; i++ {
				block = types.NewBlock(
					&types.Header{
						Number:     big.NewInt(i),
						ParentHash: blocks[i-1].Hash(),
						Root:       common.StringToHash("1234567890" + strconv.Itoa(int(i))),
					}, []*types.Transaction{tx}, nil, []*types.Receipt{receipt})
				blocks[i] = block
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

			var num *big.Int
			mockEthClient.On("BlockByNumber", mock.Anything, num).Return(blocks[15], nil).Once()

			mockEthClient.On("TransactionReceipt", mock.Anything, tx.Hash()).Return(receipt, nil).Times(9)
			var recvCh chan<- *types.Header
			recvCh = ch
			mockEthClient.On("SubscribeNewHead", mock.Anything, recvCh).Return(nil, nil).Once()

			go func() {
				ch <- blocks[18].Header()
				ch <- blocks[19].Header()
				time.Sleep(time.Second)
				cancel()
			}()

			err := idx.Listen(ctx, ch, true)
			Expect(err).Should(Equal(context.Canceled))
		})

		Context("got an old block", func() {
			It("discards the old block", func() {
				// local state has block 10,
				// initial sync blocks 11-15 from ethereum
				// receive block 13 from header channel and discards it
				blocks := make([]*types.Block, 20)
				tx := types.NewTransaction(0, common.Address{}, common.Big0, 0, common.Big0, []byte{})
				receipt := types.NewReceipt([]byte{}, false, 0)
				block := types.NewBlock(
					&types.Header{
						Number: big.NewInt(10),
						Root:   common.StringToHash("1234567890" + strconv.Itoa(int(10))),
					}, []*types.Transaction{tx}, nil, []*types.Receipt{receipt})
				blocks[10] = block
				mockStoreManager.On("LatestHeader").Return(&model.Header{
					Number: 10,
					Hash:   block.Hash().Bytes(),
				}, nil).Once()
				mockStoreManager.On("LatestStateBlock").Return(&model.StateBlock{
					Number: 10,
				}, nil).Once()

				for i := int64(11); i <= 15; i++ {
					block = types.NewBlock(
						&types.Header{
							Number:     big.NewInt(i),
							ParentHash: blocks[i-1].Hash(),
							Root:       common.StringToHash("1234567890" + strconv.Itoa(int(i))),
						}, []*types.Transaction{tx}, nil, []*types.Receipt{receipt})
					blocks[i] = block
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

				var num *big.Int
				mockEthClient.On("BlockByNumber", mock.Anything, num).Return(blocks[15], nil).Once()

				mockEthClient.On("TransactionReceipt", mock.Anything, tx.Hash()).Return(receipt, nil).Times(5)
				var recvCh chan<- *types.Header
				recvCh = ch
				mockEthClient.On("SubscribeNewHead", mock.Anything, recvCh).Return(nil, nil).Once()

				go func() {
					ch <- blocks[13].Header()
					time.Sleep(time.Second)
					cancel()
				}()

				err := idx.Listen(ctx, ch, true)
				Expect(err).Should(Equal(context.Canceled))
			})
		})

		Context("with something wrong", func() {
			It("failed to subscribe new head", func() {
				blocks := make([]*types.Block, 20)
				tx := types.NewTransaction(0, common.Address{}, common.Big0, 0, common.Big0, []byte{})
				receipt := types.NewReceipt([]byte{}, false, 0)
				block := types.NewBlock(
					&types.Header{
						Number: big.NewInt(10),
						Root:   common.StringToHash("1234567890" + strconv.Itoa(int(10))),
					}, []*types.Transaction{tx}, nil, []*types.Receipt{receipt})
				blocks[10] = block
				mockStoreManager.On("LatestHeader").Return(&model.Header{
					Number: 10,
					Hash:   block.Hash().Bytes(),
				}, nil).Once()
				mockStoreManager.On("LatestStateBlock").Return(&model.StateBlock{
					Number: 10,
				}, nil).Once()

				for i := int64(11); i <= 15; i++ {
					block = types.NewBlock(
						&types.Header{
							Number:     big.NewInt(i),
							ParentHash: blocks[i-1].Hash(),
							Root:       common.StringToHash("1234567890" + strconv.Itoa(int(i))),
						}, []*types.Transaction{tx}, nil, []*types.Receipt{receipt})
					blocks[i] = block
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

				var num *big.Int
				mockEthClient.On("BlockByNumber", mock.Anything, num).Return(blocks[15], nil).Once()

				mockEthClient.On("TransactionReceipt", mock.Anything, tx.Hash()).Return(receipt, nil).Times(5)
				var recvCh chan<- *types.Header
				recvCh = ch
				mockEthClient.On("SubscribeNewHead", mock.Anything, recvCh).Return(nil, unknownErr).Once()

				err := idx.Listen(ctx, ch, true)
				Expect(err).Should(Equal(unknownErr))
			})

			It("failed to insert block to db", func() {
				tx := types.NewTransaction(0, common.Address{}, common.Big0, 0, common.Big0, []byte{})
				receipt := types.NewReceipt([]byte{}, false, 0)
				block := types.NewBlock(
					&types.Header{
						Number: big.NewInt(10),
						Root:   common.StringToHash("1234567890" + strconv.Itoa(int(10))),
					}, []*types.Transaction{tx}, nil, []*types.Receipt{receipt})
				mockStoreManager.On("LatestHeader").Return(&model.Header{
					Number: 10,
					Hash:   block.Hash().Bytes(),
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

				block = types.NewBlock(
					&types.Header{
						Number:     big.NewInt(11),
						ParentHash: block.Hash(),
					}, []*types.Transaction{tx}, nil, []*types.Receipt{receipt})
				mockEthClient.On("BlockByNumber", mock.Anything, big.NewInt(11)).Return(block, nil).Once()
				mockEthClient.On("TransactionReceipt", mock.Anything, tx.Hash()).Return(receipt, nil).Once()
				mockStoreManager.On("InsertBlock", block, []*types.Receipt{receipt}).Return(unknownErr).Once()

				err := idx.Listen(ctx, ch, true)
				Expect(err).Should(Equal(unknownErr))
			})

			It("failed to get transaction receipt", func() {
				tx := types.NewTransaction(0, common.Address{}, common.Big0, 0, common.Big0, []byte{})
				receipt := types.NewReceipt([]byte{}, false, 0)
				block := types.NewBlock(
					&types.Header{
						Number: big.NewInt(10),
						Root:   common.StringToHash("1234567890" + strconv.Itoa(int(10))),
					}, []*types.Transaction{tx}, nil, []*types.Receipt{receipt})
				mockStoreManager.On("LatestHeader").Return(&model.Header{
					Number: 10,
					Hash:   block.Hash().Bytes(),
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

				block = types.NewBlock(
					&types.Header{
						Number:     big.NewInt(11),
						ParentHash: block.Hash(),
					}, []*types.Transaction{tx}, nil, []*types.Receipt{receipt})
				mockEthClient.On("BlockByNumber", mock.Anything, big.NewInt(11)).Return(block, nil).Once()
				mockEthClient.On("TransactionReceipt", mock.Anything, tx.Hash()).Return(nil, unknownErr).Once()

				err := idx.Listen(ctx, ch, true)
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

				err := idx.Listen(ctx, ch, true)
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
				err := idx.Listen(ctx, ch, true)
				Expect(err).Should(Equal(unknownErr))
			})

			It("failed to get state block", func() {
				mockStoreManager.On("LatestHeader").Return(&model.Header{
					Number: 10,
				}, nil).Once()
				mockStoreManager.On("LatestStateBlock").Return(nil, unknownErr).Once()
				err := idx.Listen(ctx, ch, true)
				Expect(err).Should(Equal(unknownErr))
			})

			It("failed to get latest header", func() {
				mockStoreManager.On("LatestHeader").Return(nil, unknownErr).Once()
				err := idx.Listen(ctx, ch, true)
				Expect(err).Should(Equal(unknownErr))
			})
		})
	})

	Context("Listen() with Reorg", func() {
		ctx, cancel := context.WithCancel(context.Background())
		ch := make(chan *types.Header)
		unknownErr := errors.New("unknown error")

		It("should be ok", func() {
			// local state has block 10,
			// initial sync blocks 11-15 from ethereum
			// receive 18, 19 blocks from header channel
			// when receiving block 18, we found chain was reorg'ed at block 15

			// set up old blocks
			blocks := make([]*types.Block, 20)
			tx := types.NewTransaction(0, common.Address{}, common.Big0, 0, common.Big0, []byte{})
			receipt := types.NewReceipt([]byte{}, false, 0)
			blocks[10] = types.NewBlock(
				&types.Header{
					Number: big.NewInt(10),
					Root:   common.StringToHash("1234567890" + strconv.Itoa(int(10))),
				}, []*types.Transaction{tx}, nil, []*types.Receipt{receipt})
			parentHash := common.StringToHash("123456789012345678901234567890")
			for i := int64(10); i <= 17; i++ {
				blocks[i] = types.NewBlock(
					&types.Header{
						Number:     big.NewInt(i),
						ParentHash: parentHash,
						Root:       common.StringToHash("1234567890" + strconv.Itoa(int(i))),
					}, []*types.Transaction{tx}, nil, []*types.Receipt{receipt})
				parentHash = blocks[i].Hash()
			}
			// set up new blocks
			newBlocks := make([]*types.Block, 20)
			copy(newBlocks, blocks)
			newTx := types.NewTransaction(0, common.Address{19, 23}, common.Big0, 0, common.Big0, []byte{19, 23})
			parentHash = blocks[14].Hash()
			for i := int64(15); i <= 19; i++ {
				newBlocks[i] = types.NewBlock(
					&types.Header{
						Number:     big.NewInt(i),
						ParentHash: parentHash,
						Root:       common.StringToHash("1234567890" + strconv.Itoa(int(i))),
					}, []*types.Transaction{newTx}, nil, []*types.Receipt{receipt})
				parentHash = newBlocks[i].Hash()
			}

			// set expectations
			mockStoreManager.On("LatestHeader").Return(&model.Header{
				Number: 10,
				Hash:   blocks[10].Hash().Bytes(),
			}, nil).Once()
			mockStoreManager.On("LatestHeader").Return(&model.Header{
				Number: 14,
				Hash:   blocks[14].Hash().Bytes(),
			}, nil).Once()
			mockStoreManager.On("LatestStateBlock").Return(&model.StateBlock{
				Number: 10,
			}, nil).Once()
			mockStoreManager.On("LatestStateBlock").Return(&model.StateBlock{
				Number: 14,
			}, nil).Once()

			var num *big.Int
			mockEthClient.On("BlockByNumber", mock.Anything, num).Return(blocks[15], nil).Once()

			for i := int64(11); i <= 19; i++ {
				if i <= 17 {
					mockEthClient.On("BlockByNumber", mock.Anything, big.NewInt(i)).Return(blocks[i], nil).Once()
				} else {
					mockEthClient.On("BlockByNumber", mock.Anything, big.NewInt(i)).Return(newBlocks[i], nil).Once()
				}
				// insert old blocks for 11-17
				if i <= 17 {
					mockStoreManager.On("InsertBlock", blocks[i], []*types.Receipt{receipt}).Return(nil).Once()
				}
				// insert new blocks for 15-19
				if i >= 15 {
					mockStoreManager.On("InsertBlock", newBlocks[i], []*types.Receipt{receipt}).Return(nil).Once()
				}
			}

			for i := int64(15); i <= 17; i++ {
				mockEthClient.On("BlockByHash", mock.Anything, newBlocks[i].Hash()).Return(newBlocks[i], nil).Once()
			}
			for i := int64(14); i <= 16; i++ {
				mockStoreManager.On("GetHeaderByNumber", i).Return(&model.Header{
					Number: i,
					Hash:   blocks[i].Hash().Bytes(),
				}, nil).Once()
			}
			mockStoreManager.On("DeleteDataFromBlock", int64(15)).Return(nil).Once()

			// state diff 14->15/14->16/16->17 will be called twice
			for i := int64(11); i <= 19; i++ {
				// Sometimes we cannot get account states successfully
				if i%2 == 0 {
					if i < 14 {
						dump := &state.Dump{
							Root: fmt.Sprintf("%x", blocks[i].Root()),
						}
						mockEthClient.On("ModifiedAccountStatesByNumber", mock.Anything, uint64(i-2), uint64(i)).Return(dump, nil).Once()
						mockStoreManager.On("UpdateState", blocks[i], dump).Return(nil).Once()
					} else if i >= 15 && i <= 17 {
						dump := &state.Dump{
							Root: fmt.Sprintf("%x", blocks[i].Root()),
						}
						newDump := &state.Dump{
							Root: fmt.Sprintf("%x", newBlocks[i].Root()),
						}
						mockEthClient.On("ModifiedAccountStatesByNumber", mock.Anything, uint64(i-2), uint64(i)).Return(dump, nil).Once()
						mockStoreManager.On("UpdateState", blocks[i], dump).Return(nil).Once()
						mockEthClient.On("ModifiedAccountStatesByNumber", mock.Anything, uint64(i-2), uint64(i)).Return(newDump, nil).Once()
						mockStoreManager.On("UpdateState", newBlocks[i], newDump).Return(nil).Once()
					} else {
						newDump := &state.Dump{
							Root: fmt.Sprintf("%x", newBlocks[i].Root()),
						}
						mockEthClient.On("ModifiedAccountStatesByNumber", mock.Anything, uint64(i-2), uint64(i)).Return(newDump, nil).Once()
						mockStoreManager.On("UpdateState", newBlocks[i], newDump).Return(nil).Once()
					}
				} else {
					freq := 1
					if i == 15 || i == 17 {
						freq = 2
					}
					mockEthClient.On("ModifiedAccountStatesByNumber", mock.Anything, uint64(i-1), uint64(i)).Return(nil, unknownErr).Times(freq)
				}
			}
			mockEthClient.On("TransactionReceipt", mock.Anything, tx.Hash()).Return(receipt, nil).Times(7)
			mockEthClient.On("TransactionReceipt", mock.Anything, newTx.Hash()).Return(receipt, nil).Times(5)

			var recvCh chan<- *types.Header
			recvCh = ch
			mockEthClient.On("SubscribeNewHead", mock.Anything, recvCh).Return(nil, nil).Once()

			go func() {
				ch <- newBlocks[18].Header()
				ch <- newBlocks[19].Header()
				time.Sleep(time.Second)
				cancel()
			}()

			err := idx.Listen(ctx, ch, true)
			Expect(err).Should(Equal(context.Canceled))
		})
	})
})

func TestIndexer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Indexer Test")
}
