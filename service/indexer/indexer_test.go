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
	clientMocks "github.com/getamis/eth-indexer/client/mocks"
	"github.com/getamis/eth-indexer/model"
	"github.com/getamis/eth-indexer/store"
	storeMocks "github.com/getamis/eth-indexer/store/mocks"
	"github.com/jinzhu/gorm"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

type testSub struct {
	mychan chan error
}

func (m *testSub) Err() <-chan error {
	return m.mychan
}

func (m *testSub) Unsubscribe() {
	return
}

var _ = Describe("Indexer Test", func() {
	var (
		mockSub          *testSub
		mockEthClient    *clientMocks.EthClient
		mockStoreManager *storeMocks.Manager
		idx              *indexer
		nilDirtyDump     *state.DirtyDump
	)
	BeforeEach(func() {
		mockSub = &testSub{make(chan error)}
		mockStoreManager = new(storeMocks.Manager)
		mockEthClient = new(clientMocks.EthClient)
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
			numbers := []int64{1, 2}
			ethAddresses := []common.Address{common.HexToAddress(addresses[0]), common.HexToAddress(addresses[1])}
			mockStoreManager.On("Init").Return(nil).Once()
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
			It("failed to init store manager", func() {
				addresses := []string{"0x1234567890123456789012345678901234567890", "0x1234567890123456789012345678901234567891"}
				numbers := []int64{1, 2}
				ethAddresses := []common.Address{common.HexToAddress(addresses[0]), common.HexToAddress(addresses[1])}
				mockStoreManager.On("Init").Return(unknownErr).Once()
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
				Expect(err).Should(Equal(unknownErr))
			})

			It("failed to insert ERC20", func() {
				addresses := []string{"0x1234567890123456789012345678901234567890"}
				numbers := []int64{1}
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
				numbers := []int64{1}
				ethAddresses := []common.Address{common.HexToAddress(addresses[0])}
				// The first erc20 is not found
				mockStoreManager.On("FindERC20", ethAddresses[0]).Return(nil, gorm.ErrRecordNotFound).Once()
				mockEthClient.On("GetERC20", ctx, ethAddresses[0], int64(numbers[0])).Return(nil, unknownErr).Once()
				err := idx.Init(ctx, addresses, numbers)
				Expect(err).Should(Equal(unknownErr))
			})

			It("failed to find ERC20", func() {
				addresses := []string{"0x1234567890123456789012345678901234567890"}
				numbers := []int64{1}
				ethAddresses := []common.Address{common.HexToAddress(addresses[0])}
				// The first erc20 is not found
				mockStoreManager.On("FindERC20", ethAddresses[0]).Return(nil, unknownErr).Once()
				err := idx.Init(ctx, addresses, numbers)
				Expect(err).Should(Equal(unknownErr))
			})

			It("inconsistent length between addresses and block numbers", func() {
				addresses := []string{"0x1234567890123456789012345678901234567890"}
				numbers := []int64{1, 2}
				err := idx.Init(ctx, addresses, numbers)
				Expect(err).Should(Equal(ErrInconsistentLength))
			})
		})
	})

	Context("insertTd()", func() {
		It("should be ok", func() {
			ctx := context.Background()

			difficultyStr := "11111111111111111111111111111111111111111111111111111111"
			expTD, _ := new(big.Int).SetString("22222222222222222222222222222222222222222222222222222222", 10)
			difficulty, _ := new(big.Int).SetString(difficultyStr, 10)
			block := types.NewBlockWithHeader(&types.Header{
				ParentHash: common.HexToHash("1234567890"),
				Difficulty: difficulty,
				Number:     big.NewInt(100),
			})
			mockStoreManager.On("GetTd", block.ParentHash().Bytes()).Return(&model.TotalDifficulty{
				Hash: block.ParentHash().Bytes(),
				Td:   difficultyStr,
			}, nil).Once()
			mockStoreManager.On("InsertTd", block, expTD).Return(nil).Once()
			td, err := idx.insertTd(ctx, block)
			Expect(td).Should(Equal(expTD))
			Expect(err).Should(BeNil())
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

				parent := block.ParentHash().Bytes()

				// Cannot get TD from DB, get it from etherem
				if i == 1 {
					mockStoreManager.On("GetTd", parent).Return(nil, gorm.ErrRecordNotFound).Once()
					mockEthClient.On("GetTotalDifficulty", mock.Anything, block.ParentHash()).Return(big.NewInt(i), nil).Once()
				} else {
					mockStoreManager.On("GetTd", parent).Return(&model.TotalDifficulty{
						i - 1, parent, strconv.Itoa(int(i))}, nil).Once()
				}
				mockStoreManager.On("InsertTd", block, big.NewInt(i+1)).Return(nil).Once()
				mockEthClient.On("BlockByNumber", mock.Anything, big.NewInt(i)).Return(block, nil).Once()
				mockEthClient.On("ModifiedAccountStatesByNumber", mock.Anything, uint64(i)).Return(nil, nil).Once()
				mockEthClient.On("TransactionReceipts", mock.Anything, block.Transactions()).Return([]*types.Receipt{receipt}, nil).Once()
				mockStoreManager.On("UpdateBlocks", []*types.Block{block}, [][]*types.Receipt{{receipt}}, []*state.DirtyDump{nilDirtyDump}, store.ModeForceSync).Return(nil).Once()
			}

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
					parent := block.ParentHash().Bytes()
					mockStoreManager.On("GetTd", parent).Return(&model.TotalDifficulty{
						i - i, parent, strconv.Itoa(int(i - 1))}, nil).Once()
					mockStoreManager.On("InsertTd", block, big.NewInt(i)).Return(nil).Once()
					mockEthClient.On("TransactionReceipts", mock.Anything, blocks[i].Transactions()).Return([]*types.Receipt{receipt}, nil).Once()
					mockEthClient.On("ModifiedAccountStatesByNumber", mock.Anything, uint64(i)).Return(nil, nil).Once()
					mockStoreManager.On("UpdateBlocks", []*types.Block{block}, [][]*types.Receipt{{receipt}}, []*state.DirtyDump{nilDirtyDump}, store.ModeSync).Return(nil).Once()
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

				var recvCh chan<- *types.Header
				recvCh = ch
				mockEthClient.On("SubscribeNewHead", mock.Anything, recvCh).Return(mockSub, nil).Once()

				go func() {
					ch <- blocks[18].Header()
					ch <- blocks[19].Header()
					time.Sleep(time.Second)
					cancel()
				}()

				err := idx.Listen(ctx, ch)
				Expect(err).Should(Equal(context.Canceled))
			})

			It("empty database", func() {
				ctx, cancel := context.WithCancel(context.Background())
				// init state, there is no data stored.
				// receive 19 block
				blocks := make([]*types.Block, 20)
				tx := types.NewTransaction(0, common.Address{}, common.Big0, 0, common.Big0, []byte{})
				receipt := types.NewReceipt([]byte{}, false, 0)
				block := types.NewBlock(
					&types.Header{
						Number:     big.NewInt(0),
						Root:       common.StringToHash("1234567890" + strconv.Itoa(int(10))),
						Difficulty: big.NewInt(1),
					}, []*types.Transaction{tx}, nil, []*types.Receipt{receipt})
				blocks[0] = block
				mockEthClient.On("BlockByNumber", mock.Anything, big.NewInt(0)).Return(block, nil).Once()
				mockEthClient.On("TransactionReceipts", mock.Anything, block.Transactions()).Return([]*types.Receipt{receipt}, nil).Once()
				mockStoreManager.On("InsertTd", block, big.NewInt(1)).Return(nil).Once()
				mockEthClient.On("DumpBlock", mock.Anything, int64(0)).Return(&state.Dump{
					Root: "root",
					Accounts: map[string]state.DumpAccount{
						"0001": {
							Balance: "10000",
						},
						"0002": {
							Balance: "20000",
						},
					},
				}, nil).Once()
				mockStoreManager.On("UpdateBlocks", []*types.Block{block}, [][]*types.Receipt{{receipt}}, []*state.DirtyDump{{
					Root: "root",
					Accounts: map[string]state.DirtyDumpAccount{
						"0001": {
							Balance: "10000",
						},
						"0002": {
							Balance: "20000",
						},
					},
				}}, store.ModeSync).Return(nil).Once()

				for i := int64(1); i <= 19; i++ {
					block = types.NewBlock(
						&types.Header{
							Number:     big.NewInt(i),
							ParentHash: blocks[i-1].Hash(),
							Root:       common.StringToHash("1234567890" + strconv.Itoa(int(i))),
							Difficulty: big.NewInt(1),
						}, []*types.Transaction{tx}, nil, []*types.Receipt{receipt})
					blocks[i] = block
					mockEthClient.On("BlockByNumber", mock.Anything, big.NewInt(i)).Return(block, nil).Once()
					parent := block.ParentHash().Bytes()
					mockStoreManager.On("GetTd", parent).Return(&model.TotalDifficulty{
						i - i, parent, strconv.Itoa(int(i))}, nil).Once()
					mockStoreManager.On("InsertTd", block, big.NewInt(i+1)).Return(nil).Once()
					mockEthClient.On("TransactionReceipts", mock.Anything, blocks[i].Transactions()).Return([]*types.Receipt{receipt}, nil).Once()
					mockEthClient.On("ModifiedAccountStatesByNumber", mock.Anything, uint64(i)).Return(nil, nil).Once()
					mockStoreManager.On("UpdateBlocks", []*types.Block{block}, [][]*types.Receipt{{receipt}}, []*state.DirtyDump{nilDirtyDump}, store.ModeSync).Return(nil).Once()
				}

				mockStoreManager.On("LatestHeader").Return(nil, gorm.ErrRecordNotFound).Once()
				var recvCh chan<- *types.Header
				recvCh = ch
				mockEthClient.On("SubscribeNewHead", mock.Anything, recvCh).Return(mockSub, nil).Once()

				go func() {
					ch <- blocks[19].Header()
					time.Sleep(time.Second)
					cancel()
				}()

				err := idx.Listen(ctx, ch)
				Expect(err).Should(Equal(context.Canceled))
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
					parent := block.ParentHash().Bytes()
					mockStoreManager.On("GetTd", parent).Return(&model.TotalDifficulty{
						i - i, parent, strconv.Itoa(int(i - 1))}, nil).Once()
					mockStoreManager.On("InsertTd", block, big.NewInt(i)).Return(nil).Once()
					mockEthClient.On("TransactionReceipts", mock.Anything, blocks[i].Transactions()).Return([]*types.Receipt{receipt}, nil).Once()
					mockEthClient.On("ModifiedAccountStatesByNumber", mock.Anything, uint64(i)).Return(nil, nil).Once()
					mockStoreManager.On("UpdateBlocks", []*types.Block{block}, [][]*types.Receipt{{receipt}}, []*state.DirtyDump{nilDirtyDump}, store.ModeSync).Return(nil).Once()
				}
				mockStoreManager.On("LatestHeader").Return(&model.Header{
					Number: 10,
					Hash:   blocks[10].Hash().Bytes(),
				}, nil).Once()
				mockStoreManager.On("GetTd", blocks[10].Hash().Bytes()).Return(&model.TotalDifficulty{
					10, block.Hash().Bytes(), strconv.Itoa(10)}, nil).Once()

				mockStoreManager.On("LatestHeader").Return(&model.Header{
					Number: 15,
					Hash:   blocks[15].Hash().Bytes(),
				}, nil).Once()
				mockStoreManager.On("GetTd", blocks[15].Hash().Bytes()).Return(&model.TotalDifficulty{
					15, block.Hash().Bytes(), strconv.Itoa(15)}, nil).Once()

				mockEthClient.On("BlockByNumber", mock.Anything, big.NewInt(13)).Return(blocks[13], nil).Once()
				mockStoreManager.On("GetHeaderByNumber", int64(12)).Return(&model.Header{
					Number: 12,
					Hash:   blocks[12].Hash().Bytes(),
				}, nil).Once()
				mockStoreManager.On("GetTd", blocks[13].ParentHash().Bytes()).Return(&model.TotalDifficulty{
					13, block.Hash().Bytes(), strconv.Itoa(13)}, nil).Once()

				var recvCh chan<- *types.Header
				recvCh = ch
				mockEthClient.On("SubscribeNewHead", mock.Anything, recvCh).Return(mockSub, nil).Once()

				go func() {
					ch <- blocks[15].Header()
					ch <- blocks[13].Header()
					time.Sleep(time.Second)
					cancel()
				}()

				err := idx.Listen(ctx, ch)
				Expect(err).Should(Equal(context.Canceled))
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
				mockEthClient.On("SubscribeNewHead", mock.Anything, recvCh).Return(mockSub, nil).Once()

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
				parent := block.ParentHash().Bytes()
				mockStoreManager.On("GetTd", parent).Return(&model.TotalDifficulty{
					10, parent, strconv.Itoa(10)}, nil).Once()
				mockStoreManager.On("InsertTd", block, big.NewInt(11)).Return(unknownErr).Once()

				var recvCh chan<- *types.Header
				recvCh = ch
				mockEthClient.On("SubscribeNewHead", mock.Anything, recvCh).Return(mockSub, nil).Once()

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
				mockEthClient.On("TransactionReceipts", mock.Anything, block.Transactions()).Return([]*types.Receipt{receipt}, nil).Once()
				mockEthClient.On("ModifiedAccountStatesByNumber", mock.Anything, uint64(11)).Return(nil, nil).Once()
				parent := block.ParentHash().Bytes()
				mockStoreManager.On("GetTd", parent).Return(&model.TotalDifficulty{
					10, parent, strconv.Itoa(10)}, nil).Once()
				mockStoreManager.On("InsertTd", block, big.NewInt(11)).Return(nil).Once()
				mockStoreManager.On("UpdateBlocks", []*types.Block{block}, [][]*types.Receipt{{receipt}}, []*state.DirtyDump{nilDirtyDump}, store.ModeSync).Return(unknownErr).Once()

				var recvCh chan<- *types.Header
				recvCh = ch
				mockEthClient.On("SubscribeNewHead", mock.Anything, recvCh).Return(mockSub, nil).Once()

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
				parent := block.ParentHash().Bytes()
				mockStoreManager.On("GetTd", parent).Return(&model.TotalDifficulty{
					10, parent, strconv.Itoa(10)}, nil).Once()
				mockStoreManager.On("InsertTd", block, big.NewInt(11)).Return(nil).Once()
				mockEthClient.On("TransactionReceipts", mock.Anything, block.Transactions()).Return(nil, unknownErr).Once()

				var recvCh chan<- *types.Header
				recvCh = ch
				mockEthClient.On("SubscribeNewHead", mock.Anything, recvCh).Return(mockSub, nil).Once()

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
				parent := block.ParentHash().Bytes()
				mockStoreManager.On("GetTd", parent).Return(&model.TotalDifficulty{
					10, parent, strconv.Itoa(10)}, nil).Once()
				mockStoreManager.On("InsertTd", block, big.NewInt(11)).Return(nil).Once()
				mockEthClient.On("TransactionReceipts", mock.Anything, block.Transactions()).Return([]*types.Receipt{receipt}, nil).Once()
				mockEthClient.On("ModifiedAccountStatesByNumber", mock.Anything, uint64(11)).Return(nil, unknownErr).Once()

				var recvCh chan<- *types.Header
				recvCh = ch
				mockEthClient.On("SubscribeNewHead", mock.Anything, recvCh).Return(mockSub, nil).Once()

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
				mockEthClient.On("SubscribeNewHead", mock.Anything, recvCh).Return(mockSub, nil).Once()

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
				mockEthClient.On("SubscribeNewHead", mock.Anything, recvCh).Return(mockSub, nil).Once()

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

			It("subscribe error", func() {
				subError := errors.New("client is closed")
				var recvCh chan<- *types.Header
				recvCh = ch
				mockEthClient.On("SubscribeNewHead", mock.Anything, recvCh).Return(mockSub, nil).Once()

				go func() {
					mockSub.mychan <- subError
				}()
				err := idx.Listen(ctx, ch)
				Expect(err).Should(Equal(subError))
			})
		})
	})

	Context("Listen() with Reorg", func() {
		ctx, cancel := context.WithCancel(context.Background())
		ch := make(chan *types.Header)

		It("should be ok", func() {
			// local state has block 10,
			// receive 18 blocks from header channel
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

			// receiving the first header, syncing from 16-18
			mockStoreManager.On("GetTd", blocks[15].Hash().Bytes()).Return(&model.TotalDifficulty{
				15, blocks[15].Hash().Bytes(), strconv.Itoa(15)}, nil).Once()

			// expectations for eth client
			for i := int64(16); i <= 18; i++ {
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
				parent := blocks[i].ParentHash().Bytes()
				mockStoreManager.On("GetTd", parent).Return(&model.TotalDifficulty{
					i - 1, parent, strconv.Itoa(int(i - 1))}, nil).Once()
				mockStoreManager.On("InsertTd", blocks[i], big.NewInt(i)).Return(nil).Once()
			}

			// insert new Tds for 15-19, each block has TD of 5
			prevTd := int64(14)
			// calculating TD for the new blocks 15-16
			mockStoreManager.On("GetTd", blocks[14].Hash().Bytes()).Return(&model.TotalDifficulty{
				14, blocks[14].Hash().Bytes(), strconv.Itoa(14)}, nil).Once()

			for i := int64(15); i <= 18; i++ {
				td := prevTd + 5*(i-14)
				parent := newBlocks[i].ParentHash().Bytes()
				mockStoreManager.On("GetTd", parent).Return(&model.TotalDifficulty{
					i - 1, parent, strconv.Itoa(int(td - 5))}, nil).Once()
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
				mockEthClient.On("TransactionReceipts", mock.Anything, blocks[i].Transactions()).Return([]*types.Receipt{receipt}, nil).Once()
				mockEthClient.On("ModifiedAccountStatesByNumber", mock.Anything, uint64(i)).Return(nil, nil).Once()
				mockStoreManager.On("UpdateBlocks", []*types.Block{blocks[i]}, [][]*types.Receipt{{receipt}}, []*state.DirtyDump{nilDirtyDump}, store.ModeSync).Return(nil).Once()
			}
			// state diff for the new blocks
			for i := int64(15); i <= 18; i++ {
				mockEthClient.On("TransactionReceipts", mock.Anything, newBlocks[i].Transactions()).Return([]*types.Receipt{receipt}, nil).Once()
				mockEthClient.On("ModifiedAccountStatesByNumber", mock.Anything, uint64(i)).Return(nil, nil).Once()
			}
			mockStoreManager.On("UpdateBlocks", newBlocks[15:19], [][]*types.Receipt{{receipt}, {receipt}, {receipt}, {receipt}}, []*state.DirtyDump{nilDirtyDump, nilDirtyDump, nilDirtyDump, nilDirtyDump}, store.ModeReOrg).Return(nil).Once()

			var recvCh chan<- *types.Header
			recvCh = ch
			mockEthClient.On("SubscribeNewHead", mock.Anything, recvCh).Return(mockSub, nil).Once()

			go func() {
				ch <- newBlocks[18].Header()
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
			mockStoreManager.On("GetTd", blocks[17].Hash().Bytes()).Return(&model.TotalDifficulty{
				10, blocks[17].Hash().Bytes(), strconv.Itoa(17)}, nil).Once()

			// calculating TD for the new blocks 15-16
			mockStoreManager.On("GetTd", blocks[14].Hash().Bytes()).Return(&model.TotalDifficulty{
				14, blocks[14].Hash().Bytes(), strconv.Itoa(14)}, nil).Once()

			// insert new TDs for 15 and 16, each block has TD of 5
			mockEthClient.On("BlockByNumber", mock.Anything, big.NewInt(16)).Return(newBlocks[16], nil).Once()
			for i := int64(15); i <= 16; i++ {
				td := 14 + 5*(i-14)
				parent := newBlocks[i].ParentHash().Bytes()
				mockStoreManager.On("GetTd", parent).Return(&model.TotalDifficulty{
					i - 1, parent, strconv.Itoa(int(td - 5))}, nil).Once()
				mockStoreManager.On("InsertTd", newBlocks[i], big.NewInt(td)).Return(nil).Once()
			}

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
				mockEthClient.On("TransactionReceipts", mock.Anything, newBlocks[i].Transactions()).Return([]*types.Receipt{receipt}, nil).Once()
				mockEthClient.On("ModifiedAccountStatesByNumber", mock.Anything, uint64(i)).Return(nil, nil).Once()
			}
			mockStoreManager.On("UpdateBlocks", newBlocks[15:17], [][]*types.Receipt{{receipt}, {receipt}}, []*state.DirtyDump{nilDirtyDump, nilDirtyDump}, store.ModeReOrg).Return(nil).Once()

			var recvCh chan<- *types.Header
			recvCh = ch
			mockEthClient.On("SubscribeNewHead", mock.Anything, recvCh).Return(mockSub, nil).Once()

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
