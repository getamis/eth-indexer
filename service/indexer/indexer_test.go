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
	"math/big"
	"strconv"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/jinzhu/gorm"
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
		nilDirtyDump     *state.DirtyDump
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

	Context("Init()", func() {
		ctx := context.Background()

		It("with valid parameters", func() {
			addresses := []string{"0x1234567890123456789012345678901234567890", "0x1234567890123456789012345678901234567891"}
			numbers := []int{1, 2}
			ethAddresses := []common.Address{common.HexToAddress(addresses[0]), common.HexToAddress(addresses[1])}
			// The first erc20 is not found
			mockStoreManager.On("FindERC20", ethAddresses[0]).Return(nil, gorm.ErrRecordNotFound).Once()
			erc20 := &model.ERC20{
				Address:     ethAddresses[0].Bytes(),
				BlockNumber: int64(numbers[0]),
				Name:        "name",
				Decimals:    18,
				TotalSupply: "123",
			}
			mockEthClient.On("GetERC20", ctx, ethAddresses[0], int64(numbers[0])).Return(erc20, nil).Once()
			mockStoreManager.On("InsertERC20", erc20).Return(nil).Once()
			// The second erc20 exists
			mockStoreManager.On("FindERC20", ethAddresses[1]).Return(nil, nil).Once()
			err := idx.Init(ctx, addresses, numbers)
			Expect(err).Should(BeNil())
		})

		Context("with invalid parameters", func() {
			unknownErr := errors.New("unknown error")
			It("failed to insert ERC20", func() {
				addresses := []string{"0x1234567890123456789012345678901234567890"}
				numbers := []int{1}
				ethAddresses := []common.Address{common.HexToAddress(addresses[0])}
				// The first erc20 is not found
				mockStoreManager.On("FindERC20", ethAddresses[0]).Return(nil, gorm.ErrRecordNotFound).Once()
				erc20 := &model.ERC20{
					Address:     ethAddresses[0].Bytes(),
					BlockNumber: int64(numbers[0]),
					Name:        "name",
					Decimals:    18,
					TotalSupply: "123",
				}
				mockEthClient.On("GetERC20", ctx, ethAddresses[0], int64(numbers[0])).Return(erc20, nil).Once()
				mockStoreManager.On("InsertERC20", erc20).Return(unknownErr).Once()
				err := idx.Init(ctx, addresses, numbers)
				Expect(err).Should(Equal(unknownErr))
			})

			It("failed to get ERC20 from client", func() {
				addresses := []string{"0x1234567890123456789012345678901234567890"}
				numbers := []int{1}
				ethAddresses := []common.Address{common.HexToAddress(addresses[0])}
				// The first erc20 is not found
				mockStoreManager.On("FindERC20", ethAddresses[0]).Return(nil, gorm.ErrRecordNotFound).Once()
				mockEthClient.On("GetERC20", ctx, ethAddresses[0], int64(numbers[0])).Return(nil, unknownErr).Once()
				err := idx.Init(ctx, addresses, numbers)
				Expect(err).Should(Equal(unknownErr))
			})

			It("failed to find ERC20", func() {
				addresses := []string{"0x1234567890123456789012345678901234567890"}
				numbers := []int{1}
				ethAddresses := []common.Address{common.HexToAddress(addresses[0])}
				// The first erc20 is not found
				mockStoreManager.On("FindERC20", ethAddresses[0]).Return(nil, unknownErr).Once()
				err := idx.Init(ctx, addresses, numbers)
				Expect(err).Should(Equal(unknownErr))
			})

			It("inconsistent length between addresses and block numbers", func() {
				addresses := []string{"0x1234567890123456789012345678901234567890"}
				numbers := []int{1, 2}
				err := idx.Init(ctx, addresses, numbers)
				Expect(err).Should(Equal(ErrInconsistentLength))
			})
		})
	})
	Context("SyncToTarget()", func() {
		targetBlock := int64(19)

		It("sync to target (no sync before)", func() {
			blocks := make([]*types.Block, 20)
			tx := types.NewTransaction(0, common.Address{}, common.Big0, 0, common.Big0, []byte{})
			receipt := types.NewReceipt([]byte{}, false, 0)
			for i := int64(1); i <= targetBlock; i++ {
				block := types.NewBlock(
					&types.Header{
						Number:     big.NewInt(i),
						Root:       common.StringToHash("1234567890" + strconv.Itoa(int(i))),
						Difficulty: big.NewInt(1),
					}, []*types.Transaction{tx}, nil, []*types.Receipt{receipt})
				blocks[i] = block
				mockEthClient.On("BlockByNumber", mock.Anything, big.NewInt(i)).Return(block, nil).Once()
				mockEthClient.On("ModifiedAccountStatesByNumber", mock.Anything, uint64(i)).Return(nil, nil).Once()
				mockStoreManager.On("ForceInsertBlock", block, []*types.Receipt{receipt}, nilDirtyDump).Return(nil).Once()
			}
			mockEthClient.On("TransactionReceipt", mock.Anything, tx.Hash()).Return(receipt, nil).Times(int(targetBlock))

			err := idx.SyncToTarget(context.Background(), 1, targetBlock)
			Expect(err).Should(BeNil())
		})
	})

	Context("Listen()", func() {
		ctx, cancel := context.WithCancel(context.Background())
		ch := make(chan *types.Header)
		unknownErr := errors.New("unknown error")

		Context("nothing wrong", func() {
			It("should be ok", func() {
				ctx, cancel := context.WithCancel(context.Background())

				// local state has block 10,
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
				for i := int64(11); i <= 19; i++ {
					block = types.NewBlock(
						&types.Header{
							Number:     big.NewInt(i),
							ParentHash: blocks[i-1].Hash(),
							Root:       common.StringToHash("1234567890" + strconv.Itoa(int(i))),
							Difficulty: big.NewInt(1),
						}, []*types.Transaction{tx}, nil, []*types.Receipt{receipt})
					blocks[i] = block
					mockEthClient.On("BlockByNumber", mock.Anything, big.NewInt(i)).Return(block, nil).Once()
					mockStoreManager.On("InsertTd", block, big.NewInt(i)).Return(nil).Once()
					mockEthClient.On("ModifiedAccountStatesByNumber", mock.Anything, uint64(i)).Return(nil, nil).Once()
					mockStoreManager.On("UpdateBlock", block, []*types.Receipt{receipt}, nilDirtyDump).Return(nil).Once()
				}

				mockStoreManager.On("LatestHeader").Return(&model.Header{
					Number: 10,
					Hash:   blocks[10].Hash().Bytes(),
				}, nil).Once()
				mockStoreManager.On("GetTd", blocks[10].Hash().Bytes()).Return(&model.TotalDifficulty{
					10, blocks[10].Hash().Bytes(), strconv.Itoa(10)}, nil).Once()
				mockStoreManager.On("LatestHeader").Return(&model.Header{
					Number: 18,
					Hash:   blocks[18].Hash().Bytes(),
				}, nil).Once()
				mockStoreManager.On("GetTd", blocks[18].Hash().Bytes()).Return(&model.TotalDifficulty{
					18, blocks[18].Hash().Bytes(), strconv.Itoa(18)}, nil).Once()

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

				err := idx.Listen(ctx, ch)
				Expect(err).Should(Equal(context.Canceled))
				mockStoreManager.AssertExpectations(GinkgoT())
				mockEthClient.AssertExpectations(GinkgoT())
			})

			It("disordered blocks", func() {
				ctx, cancel := context.WithCancel(context.Background())

				// local state has block 10,
				// receive block 15 from header channel
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
				for i := int64(11); i <= 15; i++ {
					block = types.NewBlock(
						&types.Header{
							Number:     big.NewInt(i),
							ParentHash: blocks[i-1].Hash(),
							Root:       common.StringToHash("1234567890" + strconv.Itoa(int(i))),
							Difficulty: big.NewInt(1),
						}, []*types.Transaction{tx}, nil, []*types.Receipt{receipt})
					blocks[i] = block
					mockEthClient.On("BlockByNumber", mock.Anything, big.NewInt(i)).Return(block, nil).Once()
					mockStoreManager.On("InsertTd", block, big.NewInt(i)).Return(nil).Once()
					mockEthClient.On("ModifiedAccountStatesByNumber", mock.Anything, uint64(i)).Return(nil, nil).Once()
					mockStoreManager.On("UpdateBlock", block, []*types.Receipt{receipt}, nilDirtyDump).Return(nil).Once()
				}
				mockStoreManager.On("LatestHeader").Return(&model.Header{
					Number: 10,
					Hash:   blocks[10].Hash().Bytes(),
				}, nil).Once()
				mockStoreManager.On("GetTd", blocks[10].Hash().Bytes()).Return(&model.TotalDifficulty{
					10, block.Hash().Bytes(), strconv.Itoa(10)}, nil).Once()
				mockEthClient.On("TransactionReceipt", mock.Anything, tx.Hash()).Return(receipt, nil).Times(5)

				mockStoreManager.On("LatestHeader").Return(&model.Header{
					Number: 15,
					Hash:   blocks[15].Hash().Bytes(),
				}, nil).Once()
				mockStoreManager.On("GetTd", blocks[15].Hash().Bytes()).Return(&model.TotalDifficulty{
					15, block.Hash().Bytes(), strconv.Itoa(15)}, nil).Once()

				var recvCh chan<- *types.Header
				recvCh = ch
				mockEthClient.On("SubscribeNewHead", mock.Anything, recvCh).Return(nil, nil).Once()

				// Expectations for checking reorg
				mockEthClient.On("BlockByNumber", mock.Anything, big.NewInt(13)).Return(blocks[13], nil).Once()
				mockStoreManager.On("GetHeaderByNumber", int64(12)).Return(&model.Header{
					Number: 12,
					Hash:   blocks[12].Hash().Bytes(),
				}, nil).Once()

				go func() {
					ch <- blocks[15].Header()
					ch <- blocks[13].Header()
					time.Sleep(time.Second)
					cancel()
				}()

				err := idx.Listen(ctx, ch)
				Expect(err).Should(Equal(context.Canceled))
				mockStoreManager.AssertExpectations(GinkgoT())
				mockEthClient.AssertExpectations(GinkgoT())
			})
		})

		Context("with something wrong", func() {
			It("failed to subscribe new head", func() {
				var recvCh chan<- *types.Header
				recvCh = ch
				mockEthClient.On("SubscribeNewHead", mock.Anything, recvCh).Return(nil, unknownErr).Once()

				err := idx.Listen(ctx, ch)
				Expect(err).Should(Equal(unknownErr))
			})

			It("failed to get TD", func() {
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
				mockStoreManager.On("GetTd", block.Hash().Bytes()).Return(&model.TotalDifficulty{}, unknownErr).Once()

				var recvCh chan<- *types.Header
				recvCh = ch
				mockEthClient.On("SubscribeNewHead", mock.Anything, recvCh).Return(nil, nil).Once()

				go func() {
					ch <- block.Header()
					time.Sleep(time.Second)
					cancel()
				}()

				err := idx.Listen(ctx, ch)
				Expect(err).Should(Equal(unknownErr))
			})

			It("failed to write TD", func() {
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
				mockStoreManager.On("GetTd", block.Hash().Bytes()).Return(&model.TotalDifficulty{
					10, block.Hash().Bytes(), strconv.Itoa(10)}, nil).Once()

				block = types.NewBlock(
					&types.Header{
						Number:     big.NewInt(11),
						ParentHash: block.Hash(),
						Difficulty: big.NewInt(1),
					}, []*types.Transaction{tx}, nil, []*types.Receipt{receipt})
				mockEthClient.On("BlockByNumber", mock.Anything, big.NewInt(11)).Return(block, nil).Once()
				mockStoreManager.On("InsertTd", block, big.NewInt(11)).Return(unknownErr).Once()

				var recvCh chan<- *types.Header
				recvCh = ch
				mockEthClient.On("SubscribeNewHead", mock.Anything, recvCh).Return(nil, nil).Once()

				go func() {
					ch <- block.Header()
					time.Sleep(time.Second)
					cancel()
				}()

				err := idx.Listen(ctx, ch)
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
				mockStoreManager.On("GetTd", block.Hash().Bytes()).Return(&model.TotalDifficulty{
					10, block.Hash().Bytes(), strconv.Itoa(10)}, nil).Once()

				block = types.NewBlock(
					&types.Header{
						Number:     big.NewInt(11),
						ParentHash: block.Hash(),
						Difficulty: big.NewInt(1),
					}, []*types.Transaction{tx}, nil, []*types.Receipt{receipt})
				mockEthClient.On("BlockByNumber", mock.Anything, big.NewInt(11)).Return(block, nil).Once()
				mockEthClient.On("TransactionReceipt", mock.Anything, tx.Hash()).Return(receipt, nil).Once()
				mockEthClient.On("ModifiedAccountStatesByNumber", mock.Anything, uint64(11)).Return(nil, nil).Once()
				mockStoreManager.On("InsertTd", block, big.NewInt(11)).Return(nil).Once()
				mockStoreManager.On("UpdateBlock", block, []*types.Receipt{receipt}, nilDirtyDump).Return(unknownErr).Once()

				var recvCh chan<- *types.Header
				recvCh = ch
				mockEthClient.On("SubscribeNewHead", mock.Anything, recvCh).Return(nil, nil).Once()

				go func() {
					ch <- block.Header()
					time.Sleep(time.Second)
					cancel()
				}()

				err := idx.Listen(ctx, ch)
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
				mockStoreManager.On("GetTd", block.Hash().Bytes()).Return(&model.TotalDifficulty{
					10, block.ParentHash().Bytes(), strconv.Itoa(10)}, nil).Once()
				block = types.NewBlock(
					&types.Header{
						Number:     big.NewInt(11),
						ParentHash: block.Hash(),
						Difficulty: big.NewInt(1),
					}, []*types.Transaction{tx}, nil, []*types.Receipt{receipt})
				mockEthClient.On("BlockByNumber", mock.Anything, big.NewInt(11)).Return(block, nil).Once()
				mockStoreManager.On("InsertTd", block, big.NewInt(11)).Return(nil).Once()
				mockEthClient.On("TransactionReceipt", mock.Anything, tx.Hash()).Return(nil, unknownErr).Once()

				var recvCh chan<- *types.Header
				recvCh = ch
				mockEthClient.On("SubscribeNewHead", mock.Anything, recvCh).Return(nil, nil).Once()

				go func() {
					ch <- block.Header()
					time.Sleep(time.Second)
					cancel()
				}()

				err := idx.Listen(ctx, ch)
				Expect(err).Should(Equal(unknownErr))
			})

			It("failed to get state diff", func() {
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
				mockStoreManager.On("GetTd", block.Hash().Bytes()).Return(&model.TotalDifficulty{
					10, block.Hash().Bytes(), strconv.Itoa(10)}, nil).Once()

				block = types.NewBlock(
					&types.Header{
						Number:     big.NewInt(11),
						ParentHash: block.Hash(),
						Difficulty: big.NewInt(1),
					}, []*types.Transaction{tx}, nil, []*types.Receipt{receipt})
				mockEthClient.On("BlockByNumber", mock.Anything, big.NewInt(11)).Return(block, nil).Once()
				mockStoreManager.On("InsertTd", block, big.NewInt(11)).Return(nil).Once()
				mockEthClient.On("TransactionReceipt", mock.Anything, tx.Hash()).Return(receipt, nil).Once()
				mockEthClient.On("ModifiedAccountStatesByNumber", mock.Anything, uint64(11)).Return(nil, unknownErr).Once()

				var recvCh chan<- *types.Header
				recvCh = ch
				mockEthClient.On("SubscribeNewHead", mock.Anything, recvCh).Return(nil, nil).Once()

				go func() {
					ch <- block.Header()
					time.Sleep(time.Second)
					cancel()
				}()

				err := idx.Listen(ctx, ch)
				Expect(err).Should(Equal(unknownErr))
			})

			It("failed to get block by number", func() {
				tx := types.NewTransaction(0, common.Address{}, common.Big0, 0, common.Big0, []byte{})
				receipt := types.NewReceipt([]byte{}, false, 0)
				block := types.NewBlock(
					&types.Header{
						Number: big.NewInt(10),
						Root:   common.StringToHash("1234567890" + strconv.Itoa(int(10))),
					}, []*types.Transaction{tx}, nil, []*types.Receipt{receipt})
				mockStoreManager.On("LatestHeader").Return(&model.Header{
					Number: 9,
					Hash:   block.Hash().Bytes(),
				}, nil).Once()
				mockStoreManager.On("GetTd", block.Hash().Bytes()).Return(&model.TotalDifficulty{
					10, block.Hash().Bytes(), strconv.Itoa(9)}, nil).Once()
				mockEthClient.On("BlockByNumber", mock.Anything, big.NewInt(10)).Return(nil, unknownErr).Once()

				var recvCh chan<- *types.Header
				recvCh = ch
				mockEthClient.On("SubscribeNewHead", mock.Anything, recvCh).Return(nil, nil).Once()

				go func() {
					ch <- block.Header()
					time.Sleep(time.Second)
					cancel()
				}()

				err := idx.Listen(ctx, ch)
				Expect(err).Should(Equal(unknownErr))
			})

			It("failed to get latest header", func() {
				mockStoreManager.On("LatestHeader").Return(nil, unknownErr).Once()

				var recvCh chan<- *types.Header
				recvCh = ch
				mockEthClient.On("SubscribeNewHead", mock.Anything, recvCh).Return(nil, nil).Once()

				go func() {
					ch <- &types.Header{
						Number: big.NewInt(10),
					}
					time.Sleep(time.Second)
					cancel()
				}()

				err := idx.Listen(ctx, ch)
				Expect(err).Should(Equal(unknownErr))
			})
		})
	})

	Context("Listen() with Reorg", func() {
		ctx, cancel := context.WithCancel(context.Background())
		ch := make(chan *types.Header)

		It("should be ok", func() {
			// local state has block 10,
			// receive 18, 19 blocks from header channel
			// when receiving block 18, we found chain was reorg'ed at block 15

			// set up old blocks
			blocks := make([]*types.Block, 20)
			tx := types.NewTransaction(0, common.Address{}, common.Big0, 0, common.Big0, []byte{})
			receipt := types.NewReceipt([]byte{}, false, 0)
			parentHash := common.StringToHash("123456789012345678901234567890")
			for i := int64(10); i <= 17; i++ {
				blocks[i] = types.NewBlock(
					&types.Header{
						Number:     big.NewInt(i),
						ParentHash: parentHash,
						Root:       common.StringToHash("1234567890" + strconv.Itoa(int(i))),
						Difficulty: big.NewInt(1),
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
						Difficulty: big.NewInt(5),
					}, []*types.Transaction{newTx}, nil, []*types.Receipt{receipt})
				parentHash = newBlocks[i].Hash()
			}

			// set expectations
			// when receiving the first header from ethereum
			mockStoreManager.On("LatestHeader").Return(&model.Header{
				Number: 15,
				Hash:   blocks[15].Hash().Bytes(),
			}, nil).Once()

			// when receiving the second header from ethereum
			mockStoreManager.On("LatestHeader").Return(&model.Header{
				Number: 18,
				Hash:   newBlocks[18].Hash().Bytes(),
			}, nil).Once()

			// receiving the first header, syncing from 16-18
			mockStoreManager.On("GetTd", blocks[15].Hash().Bytes()).Return(&model.TotalDifficulty{
				15, blocks[15].Hash().Bytes(), strconv.Itoa(15)}, nil).Once()
			// calculating TD for the new blocks 15-18
			mockStoreManager.On("GetTd", blocks[14].Hash().Bytes()).Return(&model.TotalDifficulty{
				14, blocks[14].Hash().Bytes(), strconv.Itoa(14)}, nil).Once()
			// receiving the second header, syncing new block 19
			mockStoreManager.On("GetTd", newBlocks[18].Hash().Bytes()).Return(&model.TotalDifficulty{
				18, newBlocks[18].Hash().Bytes(), strconv.Itoa(34)}, nil).Once()

			// expectations for eth client
			for i := int64(16); i <= 19; i++ {
				if i <= 17 {
					mockEthClient.On("BlockByNumber", mock.Anything, big.NewInt(i)).Return(blocks[i], nil).Once()
				} else {
					mockEthClient.On("BlockByNumber", mock.Anything, big.NewInt(i)).Return(newBlocks[i], nil).Once()
				}
			}
			// during reorg, we query block by hash to trace the canonical chain on ethereum from 18->14
			for i := int64(17); i >= 15; i-- {
				mockEthClient.On("BlockByHash", mock.Anything, newBlocks[i].Hash()).Return(newBlocks[i], nil).Once()
			}

			// expectation for store manager
			// insert old TDs for 11-17, each block has TD of 1
			for i := int64(16); i <= 17; i++ {
				mockStoreManager.On("InsertTd", blocks[i], big.NewInt(i)).Return(nil).Once()
			}

			// insert new Tds for 15-19, each block has TD of 5
			prevTd := int64(14)
			for i := int64(15); i <= 19; i++ {
				td := prevTd + 5*(i-14)
				mockStoreManager.On("InsertTd", newBlocks[i], big.NewInt(td)).Return(nil).Once()
			}
			// during reorg tracing, we query local db headers for headers to find the common ancestor of the new and old chain
			for i := int64(14); i <= 16; i++ {
				mockStoreManager.On("GetHeaderByNumber", i).Return(&model.Header{
					Number: i,
					Hash:   blocks[i].Hash().Bytes(),
				}, nil).Once()
			}

			// expectations for querying state diff
			// state diff for the old blocks
			for i := int64(16); i <= 17; i++ {
				mockEthClient.On("ModifiedAccountStatesByNumber", mock.Anything, uint64(i)).Return(nil, nil).Once()
				mockStoreManager.On("UpdateBlock", blocks[i], []*types.Receipt{receipt}, nilDirtyDump).Return(nil).Once()
			}
			// state diff for the new blocks
			for i := int64(15); i <= 19; i++ {
				mockEthClient.On("ModifiedAccountStatesByNumber", mock.Anything, uint64(i)).Return(nil, nil).Once()
				mockStoreManager.On("UpdateBlock", newBlocks[i], []*types.Receipt{receipt}, nilDirtyDump).Return(nil).Once()
			}
			mockStoreManager.On("DeleteStateFromBlock", int64(15)).Return(nil).Once()
			mockEthClient.On("TransactionReceipt", mock.Anything, tx.Hash()).Return(receipt, nil).Times(2)
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

			err := idx.Listen(ctx, ch)
			Expect(err).Should(Equal(context.Canceled))
		})
	})

	Context("Listen() old block with Reorg", func() {
		ctx, cancel := context.WithCancel(context.Background())
		ch := make(chan *types.Header)

		It("should be ok", func() {
			// local state has block 10,
			// receive 16 from header channel and found chain was reorg'ed at block 15

			// set up old blocks
			blocks := make([]*types.Block, 20)
			tx := types.NewTransaction(0, common.Address{}, common.Big0, 0, common.Big0, []byte{})
			receipt := types.NewReceipt([]byte{}, false, 0)
			parentHash := common.StringToHash("123456789012345678901234567890")
			for i := int64(10); i <= 17; i++ {
				blocks[i] = types.NewBlock(
					&types.Header{
						Number:     big.NewInt(i),
						ParentHash: parentHash,
						Root:       common.StringToHash("1234567890" + strconv.Itoa(int(i))),
						Difficulty: big.NewInt(1),
					}, []*types.Transaction{tx}, nil, []*types.Receipt{receipt})
				parentHash = blocks[i].Hash()
			}
			// set up new blocks
			newBlocks := make([]*types.Block, 20)
			copy(newBlocks, blocks)
			newTx := types.NewTransaction(0, common.Address{19, 23}, common.Big0, 0, common.Big0, []byte{19, 23})
			parentHash = blocks[14].Hash()
			for i := int64(15); i <= 16; i++ {
				newBlocks[i] = types.NewBlock(
					&types.Header{
						Number:     big.NewInt(i),
						ParentHash: parentHash,
						Root:       common.StringToHash("1234567890" + strconv.Itoa(int(i))),
						Difficulty: big.NewInt(5),
					}, []*types.Transaction{newTx}, nil, []*types.Receipt{receipt})
				parentHash = newBlocks[i].Hash()
			}

			// // set expectations
			// // at start up
			// when receiving the first header from ethereum
			mockStoreManager.On("LatestHeader").Return(&model.Header{
				Number: 17,
				Hash:   blocks[17].Hash().Bytes(),
			}, nil).Once()

			// startup sync from block 11-17
			// mockStoreManager.On("GetTd", blocks[10].Hash().Bytes()).Return(&model.TotalDifficulty{
			// 	10, blocks[10].Hash().Bytes(), strconv.Itoa(10)}, nil).Once()
			// receive header for new block 16
			mockStoreManager.On("GetTd", blocks[17].Hash().Bytes()).Return(&model.TotalDifficulty{
				10, blocks[17].Hash().Bytes(), strconv.Itoa(17)}, nil).Once()
			// calculating TD for the new blocks 15-16
			mockStoreManager.On("GetTd", blocks[14].Hash().Bytes()).Return(&model.TotalDifficulty{
				14, blocks[14].Hash().Bytes(), strconv.Itoa(14)}, nil).Once()

			// insert new TDs for 15 and 16, each block has TD of 5
			mockEthClient.On("BlockByNumber", mock.Anything, big.NewInt(16)).Return(newBlocks[16], nil).Once()
			mockStoreManager.On("InsertTd", newBlocks[15], big.NewInt(19)).Return(nil).Once()
			mockStoreManager.On("InsertTd", newBlocks[16], big.NewInt(24)).Return(nil).Once()

			// during reorg, we query block by hash to trace the canonical chain on ethereum from 16->14
			mockEthClient.On("BlockByHash", mock.Anything, newBlocks[15].Hash()).Return(newBlocks[15], nil).Once()

			// during reorg tracing, we query local db headers for headers to find the common ancestor of the new and old chain
			for i := int64(14); i <= 15; i++ {
				mockStoreManager.On("GetHeaderByNumber", i).Return(&model.Header{
					Number: i,
					Hash:   blocks[i].Hash().Bytes(),
				}, nil).Once()
			}

			// state diff for the new blocks
			for i := int64(15); i <= 16; i++ {
				mockEthClient.On("ModifiedAccountStatesByNumber", mock.Anything, uint64(i)).Return(nil, nil).Once()
				mockStoreManager.On("UpdateBlock", newBlocks[i], []*types.Receipt{receipt}, nilDirtyDump).Return(nil).Once()
			}
			mockStoreManager.On("DeleteStateFromBlock", int64(15)).Return(nil).Once()
			mockEthClient.On("TransactionReceipt", mock.Anything, newTx.Hash()).Return(receipt, nil).Times(2)

			var recvCh chan<- *types.Header
			recvCh = ch
			mockEthClient.On("SubscribeNewHead", mock.Anything, recvCh).Return(nil, nil).Once()

			go func() {
				ch <- newBlocks[16].Header()
				time.Sleep(time.Second)
				cancel()
			}()

			err := idx.Listen(ctx, ch)
			Expect(err).Should(Equal(context.Canceled))
		})
	})
})

func TestIndexer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Indexer Test")
}
