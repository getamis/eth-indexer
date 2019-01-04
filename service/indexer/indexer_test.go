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
	"database/sql"
	"errors"
	"math/big"
	"strconv"
	"testing"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/getamis/eth-indexer/client"
	clientMocks "github.com/getamis/eth-indexer/client/mocks"
	idxCommon "github.com/getamis/eth-indexer/common"
	"github.com/getamis/eth-indexer/model"
	storeMocks "github.com/getamis/eth-indexer/store/mocks"
	"github.com/getamis/hypereth/multiclient"
	"github.com/getamis/sirius/metrics"
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
		mockSubscriber   *clientMocks.Subscriber
		mockStoreManager *storeMocks.Manager
		idx              *indexer
		nilTransferLogs  []*types.TransferLog
		nilReorg         *model.Reorg
		ctx              = context.Background()
	)

	ch := make(chan<- *multiclient.Header)
	subFunc := func(ctx context.Context, c chan<- *multiclient.Header) ethereum.Subscription {
		ch = c
		return mockSub
	}

	BeforeEach(func() {
		mockSub = &testSub{make(chan error)}
		mockStoreManager = new(storeMocks.Manager)
		mockEthClient = new(clientMocks.EthClient)
		mockSubscriber = new(clientMocks.Subscriber)
		idx = &indexer{
			subscriber: mockSubscriber,
			client:     mockEthClient,
			manager:    mockStoreManager,
			newClientFunc: func(c *rpc.Client) client.EthClient {
				return mockEthClient
			},
			newBlockCounter:      metrics.NewCounter("new_block"),
			insertBlockHistogram: metrics.DefaultRegistry.NewHistogram("insert_block"),
		}
	})

	AfterEach(func() {
		mockSubscriber.AssertExpectations(GinkgoT())
		mockStoreManager.AssertExpectations(GinkgoT())
		mockEthClient.AssertExpectations(GinkgoT())
	})

	Context("SubscribeErc20Tokens()", func() {
		It("with valid parameters", func() {
			addresses := []string{"0x1234567890123456789012345678901234567890", "0x1234567890123456789012345678901234567891"}
			ethAddresses := []common.Address{common.HexToAddress(addresses[0]), common.HexToAddress(addresses[1])}
			mockStoreManager.On("Init", mock.Anything).Return(nil).Once()
			// The first erc20 is not found
			mockStoreManager.On("FindERC20", mock.Anything, ethAddresses[0]).Return(nil, sql.ErrNoRows).Once()
			erc20 := &model.ERC20{
				Address:     ethAddresses[0].Bytes(),
				BlockNumber: 0,
				Name:        "name",
				Decimals:    18,
				TotalSupply: "123",
			}
			mockEthClient.On("GetERC20", mock.Anything, ethAddresses[0]).Return(erc20, nil).Once()
			mockStoreManager.On("InsertERC20", mock.Anything, erc20).Return(nil).Once()
			// The second erc20 exists
			mockStoreManager.On("FindERC20", mock.Anything, ethAddresses[1]).Return(nil, nil).Once()
			err := idx.SubscribeErc20Tokens(ctx, addresses)
			Expect(err).Should(BeNil())
		})

		Context("with invalid parameters", func() {
			unknownErr := errors.New("unknown error")
			It("failed to init store manager", func() {
				addresses := []string{"0x1234567890123456789012345678901234567890", "0x1234567890123456789012345678901234567891"}
				ethAddresses := []common.Address{common.HexToAddress(addresses[0]), common.HexToAddress(addresses[1])}
				mockStoreManager.On("Init", mock.Anything).Return(unknownErr).Once()
				// The first erc20 is not found
				mockStoreManager.On("FindERC20", mock.Anything, ethAddresses[0]).Return(nil, sql.ErrNoRows).Once()
				erc20 := &model.ERC20{
					Address:     ethAddresses[0].Bytes(),
					BlockNumber: 0,
					Name:        "name",
					Decimals:    18,
					TotalSupply: "123",
				}
				mockEthClient.On("GetERC20", mock.Anything, ethAddresses[0]).Return(erc20, nil).Once()
				mockStoreManager.On("InsertERC20", mock.Anything, erc20).Return(nil).Once()
				// The second erc20 exists
				mockStoreManager.On("FindERC20", mock.Anything, ethAddresses[1]).Return(nil, nil).Once()
				err := idx.SubscribeErc20Tokens(ctx, addresses)
				Expect(err).Should(Equal(unknownErr))
			})

			It("failed to insert ERC20", func() {
				addresses := []string{"0x1234567890123456789012345678901234567890"}
				ethAddresses := []common.Address{common.HexToAddress(addresses[0])}
				// The first erc20 is not found
				mockStoreManager.On("FindERC20", mock.Anything, ethAddresses[0]).Return(nil, sql.ErrNoRows).Once()
				erc20 := &model.ERC20{
					Address:     ethAddresses[0].Bytes(),
					BlockNumber: 0,
					Name:        "name",
					Decimals:    18,
					TotalSupply: "123",
				}
				mockEthClient.On("GetERC20", mock.Anything, ethAddresses[0]).Return(erc20, nil).Once()
				mockStoreManager.On("InsertERC20", mock.Anything, erc20).Return(unknownErr).Once()
				err := idx.SubscribeErc20Tokens(ctx, addresses)
				Expect(err).Should(Equal(unknownErr))
			})

			It("failed to get ERC20 from client", func() {
				addresses := []string{"0x1234567890123456789012345678901234567890"}
				ethAddresses := []common.Address{common.HexToAddress(addresses[0])}
				// The first erc20 is not found
				mockStoreManager.On("FindERC20", mock.Anything, ethAddresses[0]).Return(nil, sql.ErrNoRows).Once()
				mockEthClient.On("GetERC20", mock.Anything, ethAddresses[0]).Return(nil, unknownErr).Once()
				err := idx.SubscribeErc20Tokens(ctx, addresses)
				Expect(err).Should(Equal(unknownErr))
			})

			It("failed to find ERC20", func() {
				addresses := []string{"0x1234567890123456789012345678901234567890"}
				ethAddresses := []common.Address{common.HexToAddress(addresses[0])}
				// The first erc20 is not found
				mockStoreManager.On("FindERC20", mock.Anything, ethAddresses[0]).Return(nil, unknownErr).Once()
				err := idx.SubscribeErc20Tokens(ctx, addresses)
				Expect(err).Should(Equal(unknownErr))
			})
		})
	})

	Context("insertTd()", func() {
		It("should be ok", func() {
			difficultyStr := "11111111111111111111111111111111111111111111111111111111"
			expTD, _ := new(big.Int).SetString("22222222222222222222222222222222222222222222222222222222", 10)
			difficulty, _ := new(big.Int).SetString(difficultyStr, 10)
			block := types.NewBlockWithHeader(&types.Header{
				ParentHash: common.HexToHash("1234567890"),
				Difficulty: difficulty,
				Number:     big.NewInt(100),
			})
			mockStoreManager.On("FindTd", mock.Anything, block.ParentHash().Bytes()).Return(&model.TotalDifficulty{
				Hash: block.ParentHash().Bytes(),
				Td:   difficultyStr,
			}, nil).Once()
			mockStoreManager.On("InsertTd", mock.Anything, idxCommon.TotalDifficulty(block, expTD)).Return(nil).Once()
			td, err := idx.insertTd(ctx, block)
			Expect(td).Should(Equal(expTD))
			Expect(err).Should(BeNil())
		})
	})

	Context("Listen()", func() {
		unknownErr := errors.New("unknown error")

		Context("it works fine", func() {
			It("insert blocks in sequential", func() {
				// Given local state has the block 10,
				// receive new 18 & 19 blocks from header channel

				ctx, cancel := context.WithCancel(ctx)
				blocks := make([]*types.Block, 20)
				tx := types.NewTransaction(0, common.Address{}, common.Big0, 0, common.Big0, []byte{})
				receipt := types.NewReceipt([]byte{}, false, 0)

				// the existed block 10 in database
				block := types.NewBlock(
					&types.Header{
						Number: big.NewInt(10),
						Root:   common.HexToHash("1234567890" + strconv.Itoa(int(10))),
					}, []*types.Transaction{tx}, nil, []*types.Receipt{receipt})
				blocks[10] = block
				// func addBlockMaybeReorg()
				for i := int64(11); i <= 19; i++ {
					block = types.NewBlock(
						&types.Header{
							Number:     big.NewInt(i),
							ParentHash: blocks[i-1].Hash(),
							Root:       common.HexToHash("1234567890" + strconv.Itoa(int(i))),
							Difficulty: big.NewInt(1),
						}, []*types.Transaction{tx}, nil, []*types.Receipt{receipt})
					blocks[i] = block
					mockEthClient.On("BlockByHash", mock.Anything, block.Hash()).Return(block, nil).Once()
					parent := block.ParentHash().Bytes()
					mockStoreManager.On("FindTd", mock.Anything, parent).Return(&model.TotalDifficulty{
						i - 1, parent, strconv.Itoa(int(i - 1))}, nil).Once()
					mockStoreManager.On("InsertTd", mock.Anything, idxCommon.TotalDifficulty(block, big.NewInt(i))).Return(nil).Once()
					mockEthClient.On("GetBlockReceipts", mock.Anything, blocks[i].Hash()).Return(types.Receipts{receipt}, nil).Once()
					mockEthClient.On("GetTransferLogs", mock.Anything, blocks[i].Hash()).Return(nil, nil).Once()
				}

				// deal with the new header 18,
				// blocks from 11 to 18
				// func getLocalState()
				mockStoreManager.On("FindLatestBlock", mock.Anything).Return(&model.Header{
					Number: 10,
					Hash:   blocks[10].Hash().Bytes(),
				}, nil).Once()
				mockStoreManager.On("FindTd", mock.Anything, blocks[10].Hash().Bytes()).Return(&model.TotalDifficulty{
					10, blocks[10].Hash().Bytes(), strconv.Itoa(10)}, nil).Once()
				mockEthClient.On("GetTotalDifficulty", mock.Anything, blocks[18].Hash()).Return(big.NewInt(18), nil).Once()
				var rs [][]*types.Receipt
				var ts [][]*types.TransferLog
				for i := 11; i <= 18; i++ {
					rs = append(rs, []*types.Receipt{receipt})
					ts = append(ts, nilTransferLogs)
				}
				mockStoreManager.On("UpdateBlocks", mock.Anything, mockEthClient, blocks[11:19], rs, ts, nilReorg).Return(nil).Once()

				mockStoreManager.On("UpdateBlocks", mock.Anything, mockEthClient, blocks[19:20], [][]*types.Receipt{{receipt}}, [][]*types.TransferLog{nilTransferLogs}, nilReorg).Return(nil).Once()

				mockSubscriber.On("SubscribeNewHead", mock.Anything, mock.Anything).Return(subFunc, nil).Once()

				go func() {
					time.Sleep(time.Second)
					ch <- &multiclient.Header{
						Header: blocks[18].Header(),
					}
					ch <- &multiclient.Header{
						Header: blocks[19].Header(),
					}
					time.Sleep(time.Second)
					cancel()
				}()

				err := idx.Listen(ctx, 0)
				Expect(err).Should(Equal(context.Canceled))
			})

			It("start indexer with a given block and cause block data gap in database", func() {
				// Given a empty database and a new header 19.
				// Should insert all the new blocks 15 ~ 19.
				ctx, cancel := context.WithCancel(ctx)
				// init state, there is no data stored.
				blocks := make([]*types.Block, 20)
				tx := types.NewTransaction(0, common.Address{}, common.Big0, 0, common.Big0, []byte{})
				receipt := types.NewReceipt([]byte{}, false, 0)

				// the genesis block 0
				block := types.NewBlock(
					&types.Header{
						Number:     big.NewInt(0),
						Root:       common.HexToHash("1234567890" + strconv.Itoa(int(10))),
						Difficulty: big.NewInt(1),
					}, []*types.Transaction{tx}, nil, []*types.Receipt{receipt})
				blocks[0] = block

				// func addBlockMaybeReorg()
				for i := int64(1); i <= 19; i++ {
					block = types.NewBlock(
						&types.Header{
							Number:     big.NewInt(i),
							ParentHash: blocks[i-1].Hash(),
							Root:       common.HexToHash("1234567890" + strconv.Itoa(int(i))),
							Difficulty: big.NewInt(1),
						}, []*types.Transaction{tx}, nil, []*types.Receipt{receipt})
					blocks[i] = block
					if i >= 15 {
						parent := block.ParentHash().Bytes()
						mockStoreManager.On("FindTd", mock.Anything, parent).Return(&model.TotalDifficulty{
							i - 1, parent, strconv.Itoa(int(i))}, nil).Once()

						// func insertBlocks()
						mockStoreManager.On("InsertTd", mock.Anything, idxCommon.TotalDifficulty(block, big.NewInt(i+1))).Return(nil).Once()
						mockEthClient.On("GetBlockReceipts", mock.Anything, blocks[i].Hash()).Return(types.Receipts{receipt}, nil).Once()
						mockEthClient.On("GetTransferLogs", mock.Anything, blocks[i].Hash()).Return(nil, nil).Once()
					}
				}
				for i := int64(16); i <= 19; i++ {
					mockEthClient.On("BlockByHash", mock.Anything, blocks[i].Hash()).Return(blocks[i], nil).Once()
				}
				// Check if from block exists
				mockStoreManager.On("FindLatestBlock", mock.Anything).Return(nil, sql.ErrNoRows).Once()
				mockEthClient.On("BlockByNumber", mock.Anything, big.NewInt(15)).Return(blocks[15], nil).Once()
				mockStoreManager.On("UpdateBlocks", mock.Anything, mockEthClient, []*types.Block{blocks[15]}, [][]*types.Receipt{{receipt}}, [][]*types.TransferLog{nilTransferLogs}, nilReorg).Return(nil).Once()

				mockEthClient.On("GetTotalDifficulty", mock.Anything, blocks[19].Hash()).Return(big.NewInt(19), nil).Once()
				var rs [][]*types.Receipt
				var ts [][]*types.TransferLog
				for i := 16; i <= 19; i++ {
					rs = append(rs, []*types.Receipt{receipt})
					ts = append(ts, nilTransferLogs)
				}
				mockStoreManager.On("UpdateBlocks", mock.Anything, mockEthClient, blocks[16:20], rs, ts, nilReorg).Return(nil).Once()

				mockSubscriber.On("SubscribeNewHead", mock.Anything, mock.Anything).Return(subFunc, nil).Once()

				go func() {
					time.Sleep(time.Second)
					ch <- &multiclient.Header{
						Header: blocks[19].Header(),
					}
					time.Sleep(time.Second)
					cancel()
				}()

				err := idx.Listen(ctx, 15)
				Expect(err).Should(Equal(context.Canceled))
			})

			It("insert blocks with empty database", func() {
				// Given a empty database and a new header 19.
				// Should insert all the new blocks 0 ~ 19.

				ctx, cancel := context.WithCancel(ctx)
				// init state, there is no data stored.
				blocks := make([]*types.Block, 20)
				tx := types.NewTransaction(0, common.Address{}, common.Big0, 0, common.Big0, []byte{})
				receipt := types.NewReceipt([]byte{}, false, 0)

				// the genesis block 0
				block := types.NewBlock(
					&types.Header{
						Number:     big.NewInt(0),
						Root:       common.HexToHash("1234567890" + strconv.Itoa(int(10))),
						Difficulty: big.NewInt(1),
					}, []*types.Transaction{tx}, nil, []*types.Receipt{receipt})
				blocks[0] = block

				// func addBlockMaybeReorg()
				for i := int64(1); i <= 19; i++ {
					block = types.NewBlock(
						&types.Header{
							Number:     big.NewInt(i),
							ParentHash: blocks[i-1].Hash(),
							Root:       common.HexToHash("1234567890" + strconv.Itoa(int(i))),
							Difficulty: big.NewInt(1),
						}, []*types.Transaction{tx}, nil, []*types.Receipt{receipt})
					blocks[i] = block
					mockEthClient.On("BlockByHash", mock.Anything, block.Hash()).Return(block, nil).Once()
					parent := block.ParentHash().Bytes()
					mockStoreManager.On("FindTd", mock.Anything, parent).Return(&model.TotalDifficulty{
						i - 1, parent, strconv.Itoa(int(i))}, nil).Once()

					// func insertBlocks()
					mockStoreManager.On("InsertTd", mock.Anything, idxCommon.TotalDifficulty(block, big.NewInt(i+1))).Return(nil).Once()
					mockEthClient.On("GetBlockReceipts", mock.Anything, blocks[i].Hash()).Return(types.Receipts{receipt}, nil).Once()
					mockEthClient.On("GetTransferLogs", mock.Anything, blocks[i].Hash()).Return(nil, nil).Once()
				}

				// Check if from block exists
				mockStoreManager.On("FindLatestBlock", mock.Anything).Return(nil, sql.ErrNoRows).Once()
				mockEthClient.On("BlockByNumber", mock.Anything, big.NewInt(0)).Return(blocks[0], nil).Once()
				mockStoreManager.On("InsertTd", mock.Anything, idxCommon.TotalDifficulty(blocks[0], big.NewInt(1))).Return(nil).Once()
				mockStoreManager.On("UpdateBlocks", mock.Anything, mockEthClient, []*types.Block{blocks[0]}, [][]*types.Receipt{{}}, [][]*types.TransferLog{{}}, nilReorg).Return(nil).Once()

				mockEthClient.On("GetTotalDifficulty", mock.Anything, blocks[19].Hash()).Return(big.NewInt(19), nil).Once()
				var rs [][]*types.Receipt
				var ts [][]*types.TransferLog
				for i := 1; i <= 19; i++ {
					rs = append(rs, []*types.Receipt{receipt})
					ts = append(ts, nilTransferLogs)
				}
				mockStoreManager.On("UpdateBlocks", mock.Anything, mockEthClient, blocks[1:20], rs, ts, nilReorg).Return(nil).Once()

				mockSubscriber.On("SubscribeNewHead", mock.Anything, mock.Anything).Return(subFunc, nil).Once()

				go func() {
					time.Sleep(time.Second)
					ch <- &multiclient.Header{
						Header: blocks[19].Header(),
					}
					time.Sleep(time.Second)
					cancel()
				}()

				err := idx.Listen(ctx, 0)
				Expect(err).Should(Equal(context.Canceled))
			})

			It("ignore old block", func() {
				ctx, cancel := context.WithCancel(ctx)
				blocks := make([]*types.Block, 20)
				tx := types.NewTransaction(0, common.Address{}, common.Big0, 0, common.Big0, []byte{})
				receipt := types.NewReceipt([]byte{}, false, 0)
				block := types.NewBlock(
					&types.Header{
						Number: big.NewInt(10),
						Root:   common.HexToHash("1234567890" + strconv.Itoa(int(10))),
					}, []*types.Transaction{tx}, nil, []*types.Receipt{receipt})
				blocks[10] = block
				for i := int64(11); i <= 15; i++ {
					block = types.NewBlock(
						&types.Header{
							Number:     big.NewInt(i),
							ParentHash: blocks[i-1].Hash(),
							Root:       common.HexToHash("1234567890" + strconv.Itoa(int(i))),
							Difficulty: big.NewInt(1),
						}, []*types.Transaction{tx}, nil, []*types.Receipt{receipt})
					blocks[i] = block
				}

				mockStoreManager.On("FindLatestBlock", mock.Anything).Return(&model.Header{
					Number: 15,
					Hash:   blocks[15].Hash().Bytes(),
				}, nil).Once()
				mockStoreManager.On("FindTd", mock.Anything, blocks[15].Hash().Bytes()).Return(&model.TotalDifficulty{
					15, blocks[15].Hash().Bytes(), strconv.Itoa(int(15))}, nil).Once()

				mockSubscriber.On("SubscribeNewHead", mock.Anything, mock.Anything).Return(subFunc, nil).Once()

				go func() {
					time.Sleep(time.Second)
					// Ignore old block
					ch <- &multiclient.Header{
						Header: blocks[10].Header(),
					}
					// Ignore old block
					ch <- &multiclient.Header{
						Header: blocks[14].Header(),
					}

					time.Sleep(time.Second)
					cancel()
				}()

				err := idx.Listen(ctx, 15)
				Expect(err).Should(Equal(context.Canceled))
			})

			It("ignore the blocks which have already been recorded in database", func() {
				// Given local state has the block 10.
				// Receive block 15 from header channel first, insert blocks 11 ~ 15.
				// Receive block 13 from header channel then ignore directly.

				ctx, cancel := context.WithCancel(ctx)
				blocks := make([]*types.Block, 20)
				tx := types.NewTransaction(0, common.Address{}, common.Big0, 0, common.Big0, []byte{})
				receipt := types.NewReceipt([]byte{}, false, 0)
				block := types.NewBlock(
					&types.Header{
						Number: big.NewInt(10),
						Root:   common.HexToHash("1234567890" + strconv.Itoa(int(10))),
					}, []*types.Transaction{tx}, nil, []*types.Receipt{receipt})
				blocks[10] = block
				// func addBlockMaybeReorg()
				for i := int64(11); i <= 15; i++ {
					block = types.NewBlock(
						&types.Header{
							Number:     big.NewInt(i),
							ParentHash: blocks[i-1].Hash(),
							Root:       common.HexToHash("1234567890" + strconv.Itoa(int(i))),
							Difficulty: big.NewInt(1),
						}, []*types.Transaction{tx}, nil, []*types.Receipt{receipt})
					blocks[i] = block
					mockEthClient.On("BlockByHash", mock.Anything, block.Hash()).Return(block, nil).Once()
					parent := block.ParentHash().Bytes()
					mockStoreManager.On("FindTd", mock.Anything, parent).Return(&model.TotalDifficulty{
						i - 1, parent, strconv.Itoa(int(i - 1))}, nil).Once()
					// func insertBlocks()
					mockStoreManager.On("InsertTd", mock.Anything, idxCommon.TotalDifficulty(block, big.NewInt(i))).Return(nil).Once()
					mockEthClient.On("GetBlockReceipts", mock.Anything, blocks[i].Hash()).Return(types.Receipts{receipt}, nil).Once()
					mockEthClient.On("GetTransferLogs", mock.Anything, blocks[i].Hash()).Return(nil, nil).Once()
				}
				mockStoreManager.On("FindLatestBlock", mock.Anything).Return(&model.Header{
					Number: 10,
					Hash:   blocks[10].Hash().Bytes(),
				}, nil).Once()
				mockStoreManager.On("FindTd", mock.Anything, blocks[10].Hash().Bytes()).Return(&model.TotalDifficulty{
					10, block.Hash().Bytes(), strconv.Itoa(10)}, nil).Once()

				mockEthClient.On("GetTotalDifficulty", mock.Anything, blocks[15].Hash()).Return(big.NewInt(15), nil).Once()
				var rs [][]*types.Receipt
				var ts [][]*types.TransferLog
				for i := 11; i <= 15; i++ {
					rs = append(rs, []*types.Receipt{receipt})
					ts = append(ts, nilTransferLogs)
				}
				mockStoreManager.On("UpdateBlocks", mock.Anything, mockEthClient, blocks[11:16], rs, ts, nilReorg).Return(nil).Once()

				mockSubscriber.On("SubscribeNewHead", mock.Anything, mock.Anything).Return(subFunc, nil).Once()

				go func() {
					time.Sleep(time.Second)
					ch <- &multiclient.Header{
						Header: blocks[15].Header(),
					}
					ch <- &multiclient.Header{
						Header: blocks[13].Header(),
					}
					time.Sleep(time.Second)
					cancel()
				}()

				err := idx.Listen(ctx, 0)
				Expect(err).Should(Equal(context.Canceled))
			})
		})

		Context("something goes wrong", func() {
			It("failed to FindTd()", func() {
				// Given init state has the block 10 but failed to get its total difficulty.

				tx := types.NewTransaction(0, common.Address{}, common.Big0, 0, common.Big0, []byte{})
				receipt := types.NewReceipt([]byte{}, false, 0)
				block := types.NewBlock(
					&types.Header{
						Number: big.NewInt(10),
						Root:   common.HexToHash("1234567890" + strconv.Itoa(int(10))),
					}, []*types.Transaction{tx}, nil, []*types.Receipt{receipt})

				// func getLocalState()
				mockStoreManager.On("FindLatestBlock", mock.Anything).Return(&model.Header{
					Number: 10,
					Hash:   block.Hash().Bytes(),
				}, nil).Once()

				err := idx.Listen(ctx, 100)
				Expect(err).Should(Equal(ErrDirtyDatabase))
			})

			It("failed to FindTd()", func() {
				// Given init state has the block 10 but failed to get its total difficulty.
				tx := types.NewTransaction(0, common.Address{}, common.Big0, 0, common.Big0, []byte{})
				receipt := types.NewReceipt([]byte{}, false, 0)
				block := types.NewBlock(
					&types.Header{
						Number: big.NewInt(10),
						Root:   common.HexToHash("1234567890" + strconv.Itoa(int(10))),
					}, []*types.Transaction{tx}, nil, []*types.Receipt{receipt})

				// func getLocalState()
				mockStoreManager.On("FindLatestBlock", mock.Anything).Return(&model.Header{
					Number: 10,
					Hash:   block.Hash().Bytes(),
				}, nil).Once()

				// cause error here
				mockStoreManager.On("FindTd", mock.Anything, block.Hash().Bytes()).Return(&model.TotalDifficulty{}, unknownErr).Once()

				err := idx.Listen(ctx, 0)
				Expect(err).Should(Equal(unknownErr))
			})

			It("failed to InsertTd()", func() {
				// Given init state has the block 10.
				// Received new header 11 but failed to insert total difficulty.

				ctx, cancel := context.WithCancel(ctx)
				tx := types.NewTransaction(0, common.Address{}, common.Big0, 0, common.Big0, []byte{})
				receipt := types.NewReceipt([]byte{}, false, 0)
				block := types.NewBlock(
					&types.Header{
						Number: big.NewInt(10),
						Root:   common.HexToHash("1234567890" + strconv.Itoa(int(10))),
					}, []*types.Transaction{tx}, nil, []*types.Receipt{receipt})

				// func getLocalState()
				mockStoreManager.On("FindLatestBlock", mock.Anything).Return(&model.Header{
					Number: 10,
					Hash:   block.Hash().Bytes(),
				}, nil).Once()
				mockStoreManager.On("FindTd", mock.Anything, block.Hash().Bytes()).Return(&model.TotalDifficulty{
					10, block.Hash().Bytes(), strconv.Itoa(10)}, nil).Once()

				block = types.NewBlock(
					&types.Header{
						Number:     big.NewInt(11),
						ParentHash: block.Hash(),
						Difficulty: big.NewInt(1),
					}, []*types.Transaction{tx}, nil, []*types.Receipt{receipt})

				// insert the new block 11
				// func addBlockMaybeReorg()
				mockEthClient.On("BlockByHash", mock.Anything, block.Hash()).Return(block, nil).Once()
				parent := block.ParentHash().Bytes()
				mockStoreManager.On("FindTd", mock.Anything, parent).Return(&model.TotalDifficulty{
					10, parent, strconv.Itoa(10)}, nil).Once()

				// cause error here
				mockStoreManager.On("InsertTd", mock.Anything, idxCommon.TotalDifficulty(block, big.NewInt(11))).Return(unknownErr).Once()

				mockSubscriber.On("SubscribeNewHead", mock.Anything, mock.Anything).Return(subFunc, nil).Once()

				go func() {
					time.Sleep(time.Second)
					// new header: 11
					ch <- &multiclient.Header{
						Header: block.Header(),
					}
					time.Sleep(time.Second)
					cancel()
				}()

				err := idx.Listen(ctx, 0)
				Expect(err).Should(Equal(context.Canceled))
			})

			It("failed to InsertTd()", func() {
				// Given init state has the block 10.
				// Received new header 11 but failed to insert total difficulty.

				ctx, cancel := context.WithCancel(ctx)
				tx := types.NewTransaction(0, common.Address{}, common.Big0, 0, common.Big0, []byte{})
				receipt := types.NewReceipt([]byte{}, false, 0)
				block := types.NewBlock(
					&types.Header{
						Number: big.NewInt(10),
						Root:   common.HexToHash("1234567890" + strconv.Itoa(int(10))),
					}, []*types.Transaction{tx}, nil, []*types.Receipt{receipt})

				// func getLocalState()
				mockStoreManager.On("FindLatestBlock", mock.Anything).Return(&model.Header{
					Number: 10,
					Hash:   block.Hash().Bytes(),
				}, nil).Once()
				mockStoreManager.On("FindTd", mock.Anything, block.Hash().Bytes()).Return(&model.TotalDifficulty{
					10, block.Hash().Bytes(), strconv.Itoa(10)}, nil).Once()

				block = types.NewBlock(
					&types.Header{
						Number:     big.NewInt(11),
						ParentHash: block.Hash(),
						Difficulty: big.NewInt(1),
					}, []*types.Transaction{tx}, nil, []*types.Receipt{receipt})

				// insert the new block 11
				// func addBlockMaybeReorg()
				mockEthClient.On("BlockByHash", mock.Anything, block.Hash()).Return(block, nil).Once()
				parent := block.ParentHash().Bytes()
				mockStoreManager.On("FindTd", mock.Anything, parent).Return(&model.TotalDifficulty{
					10, parent, strconv.Itoa(10)}, nil).Once()

				// cause error here
				mockStoreManager.On("InsertTd", mock.Anything, idxCommon.TotalDifficulty(block, big.NewInt(11))).Return(unknownErr).Once()

				mockSubscriber.On("SubscribeNewHead", mock.Anything, mock.Anything).Return(subFunc, nil).Once()

				go func() {
					time.Sleep(time.Second)
					// new header: 11
					ch <- &multiclient.Header{
						Header: block.Header(),
					}
					time.Sleep(time.Second)
					cancel()
				}()

				err := idx.Listen(ctx, 0)
				Expect(err).Should(Equal(context.Canceled))
			})

			It("failed to insert block to database via UpdateBlocks()", func() {
				// Given init state has the block 10.
				// Received new header 11 but failed to update the block 11.

				ctx, cancel := context.WithCancel(ctx)
				tx := types.NewTransaction(0, common.Address{}, common.Big0, 0, common.Big0, []byte{})
				receipt := types.NewReceipt([]byte{}, false, 0)

				// the existed block 10 in database
				block := types.NewBlock(
					&types.Header{
						Number: big.NewInt(10),
						Root:   common.HexToHash("1234567890" + strconv.Itoa(int(10))),
					}, []*types.Transaction{tx}, nil, []*types.Receipt{receipt})

				// func getLocalState()
				mockStoreManager.On("FindLatestBlock", mock.Anything).Return(&model.Header{
					Number: 10,
					Hash:   block.Hash().Bytes(),
				}, nil).Once()
				mockStoreManager.On("FindTd", mock.Anything, block.Hash().Bytes()).Return(&model.TotalDifficulty{
					10, block.Hash().Bytes(), strconv.Itoa(10)}, nil).Once()

				block = types.NewBlock(
					&types.Header{
						Number:     big.NewInt(11),
						ParentHash: block.Hash(),
						Difficulty: big.NewInt(1),
					}, []*types.Transaction{tx}, nil, []*types.Receipt{receipt})

				// insert the new block 11
				// func addBlockMaybeReorg()
				mockEthClient.On("BlockByHash", mock.Anything, block.Hash()).Return(block, nil).Once()
				mockEthClient.On("GetBlockReceipts", mock.Anything, block.Hash()).Return(types.Receipts{receipt}, nil).Once()
				parent := block.ParentHash().Bytes()
				mockStoreManager.On("FindTd", mock.Anything, parent).Return(&model.TotalDifficulty{
					10, parent, strconv.Itoa(10)}, nil).Once()
				mockStoreManager.On("InsertTd", mock.Anything, idxCommon.TotalDifficulty(block, big.NewInt(11))).Return(nil).Once()
				mockEthClient.On("GetTransferLogs", mock.Anything, block.Hash()).Return(nil, nil).Once()

				// cause error here
				mockStoreManager.On("UpdateBlocks", mock.Anything, mockEthClient, []*types.Block{block}, [][]*types.Receipt{{receipt}}, [][]*types.TransferLog{nilTransferLogs}, nilReorg).Return(unknownErr).Once()

				mockSubscriber.On("SubscribeNewHead", mock.Anything, mock.Anything).Return(subFunc, nil).Once()

				go func() {
					time.Sleep(time.Second)
					// New header: block 11
					ch <- &multiclient.Header{
						Header: block.Header(),
					}

					time.Sleep(time.Second)
					cancel()
				}()

				err := idx.Listen(ctx, 0)
				Expect(err).Should(Equal(context.Canceled))
			})

			It("failed to get transaction receipt via GetBlockReceipts()", func() {
				// Given init state has the block 10.
				// Received new header 11 but failed to get the receipt of block 11.

				ctx, cancel := context.WithCancel(ctx)
				tx := types.NewTransaction(0, common.Address{}, common.Big0, 0, common.Big0, []byte{})
				receipt := types.NewReceipt([]byte{}, false, 0)

				// the existed block 10 in database
				block := types.NewBlock(
					&types.Header{
						Number: big.NewInt(10),
						Root:   common.HexToHash("1234567890" + strconv.Itoa(int(10))),
					}, []*types.Transaction{tx}, nil, []*types.Receipt{receipt})

				// func getLocalState()
				mockStoreManager.On("FindLatestBlock", mock.Anything).Return(&model.Header{
					Number: 10,
					Hash:   block.Hash().Bytes(),
				}, nil).Once()
				mockStoreManager.On("FindTd", mock.Anything, block.Hash().Bytes()).Return(&model.TotalDifficulty{
					10, block.ParentHash().Bytes(), strconv.Itoa(10)}, nil).Once()

				block = types.NewBlock(
					&types.Header{
						Number:     big.NewInt(11),
						ParentHash: block.Hash(),
						Difficulty: big.NewInt(1),
					}, []*types.Transaction{tx}, nil, []*types.Receipt{receipt})

				// insert the new block 11
				// func addBlockMaybeReorg()
				mockEthClient.On("BlockByHash", mock.Anything, block.Hash()).Return(block, nil).Once()
				parent := block.ParentHash().Bytes()
				mockStoreManager.On("FindTd", mock.Anything, parent).Return(&model.TotalDifficulty{
					10, parent, strconv.Itoa(10)}, nil).Once()
				mockStoreManager.On("InsertTd", mock.Anything, idxCommon.TotalDifficulty(block, big.NewInt(11))).Return(nil).Once()

				// func getBlockData()
				// cause error here
				mockEthClient.On("GetBlockReceipts", mock.Anything, block.Hash()).Return(nil, unknownErr).Once()

				mockSubscriber.On("SubscribeNewHead", mock.Anything, mock.Anything).Return(subFunc, nil).Once()

				go func() {
					time.Sleep(time.Second)
					// new header 11
					ch <- &multiclient.Header{
						Header: block.Header(),
					}
					time.Sleep(time.Second)
					cancel()
				}()

				err := idx.Listen(ctx, 0)
				Expect(err).Should(Equal(context.Canceled))
			})

			It("failed to get block by number via BlockByHash()", func() {
				// Given init state has the block 9.
				// Received new header 11 but failed to get the block info of 10.

				ctx, cancel := context.WithCancel(ctx)
				tx := types.NewTransaction(0, common.Address{}, common.Big0, 0, common.Big0, []byte{})
				receipt := types.NewReceipt([]byte{}, false, 0)
				block := types.NewBlock(
					&types.Header{
						Number: big.NewInt(10),
						Root:   common.HexToHash("1234567890" + strconv.Itoa(int(10))),
					}, []*types.Transaction{tx}, nil, []*types.Receipt{receipt})

				// Given init state has the block 9
				// func getLocalState()
				mockStoreManager.On("FindLatestBlock", mock.Anything).Return(&model.Header{
					Number: 9,
					Hash:   block.Hash().Bytes(),
				}, nil).Once()
				mockStoreManager.On("FindTd", mock.Anything, block.Hash().Bytes()).Return(&model.TotalDifficulty{
					10, block.Hash().Bytes(), strconv.Itoa(9)}, nil).Once()

				// cause error here
				mockEthClient.On("BlockByHash", mock.Anything, block.Hash()).Return(nil, unknownErr).Once()

				mockSubscriber.On("SubscribeNewHead", mock.Anything, mock.Anything).Return(subFunc, nil).Once()

				go func() {
					time.Sleep(time.Second)
					ch <- &multiclient.Header{
						Header: block.Header(),
					}
					time.Sleep(time.Second)
					cancel()
				}()

				err := idx.Listen(ctx, 0)
				Expect(err).Should(Equal(context.Canceled))
			})

			It("failed to get latest header", func() {
				mockStoreManager.On("FindLatestBlock", mock.Anything).Return(nil, unknownErr).Once()
				err := idx.Listen(ctx, 0)
				Expect(err).Should(Equal(unknownErr))
			})

		})
	})

	Context("Listen() reorg the new blocks", func() {
		ctx, cancel := context.WithCancel(ctx)

		It("works fine", func() {
			// Given local state has the blocks 10 ~ 15,
			// received the new header 18 from header channel.
			// We found that chain was reorg'ed at block 15 (blocks 15 ~ 18 were changed)

			// set up old blocks: 11 ~ 15
			blocks := make([]*types.Block, 20)
			tx := types.NewTransaction(0, common.Address{}, common.Big0, 0, common.Big0, []byte{})
			receipt := types.NewReceipt([]byte{}, false, 0)
			parentHash := common.HexToHash("123456789012345678901234567890")
			for i := int64(10); i <= 15; i++ {
				blocks[i] = types.NewBlock(
					&types.Header{
						Number:     big.NewInt(i),
						ParentHash: parentHash,
						Root:       common.HexToHash("1234567890" + strconv.Itoa(int(i))),
						Difficulty: big.NewInt(1),
					}, []*types.Transaction{tx}, nil, []*types.Receipt{receipt})
				parentHash = blocks[i].Hash()
			}

			// set up new blocks: 15 ~ 18
			newBlocks := make([]*types.Block, 20)
			copy(newBlocks, blocks)
			newTx := types.NewTransaction(0, common.Address{19, 23}, common.Big0, 0, common.Big0, []byte{19, 23})
			// parentHash changed here
			parentHash = blocks[14].Hash()
			for i := int64(15); i <= 19; i++ {
				newBlocks[i] = types.NewBlock(
					&types.Header{
						Number:     big.NewInt(i),
						ParentHash: parentHash,
						Root:       common.HexToHash("1234567890" + strconv.Itoa(int(i))),
						Difficulty: big.NewInt(5),
					}, []*types.Transaction{newTx}, nil, []*types.Receipt{receipt})
				parentHash = newBlocks[i].Hash()
			}

			// func getLocalState()
			// set expectations
			// when receiving the first header 18, checking the latest header in database
			mockStoreManager.On("FindLatestBlock", mock.Anything).Return(&model.Header{
				Number: 15,
				Hash:   blocks[15].Hash().Bytes(),
			}, nil).Once()
			mockStoreManager.On("FindTd", mock.Anything, blocks[15].Hash().Bytes()).Return(&model.TotalDifficulty{
				15, blocks[15].Hash().Bytes(), strconv.Itoa(15)}, nil).Once()

			mockEthClient.On("GetTotalDifficulty", mock.Anything, newBlocks[18].Hash()).Return(big.NewInt(34), nil).Once()

			// during the reorg, we query block by hash to trace the canonical chain on ethereum from 17 to 15
			for i := int64(18); i >= 15; i-- {
				mockEthClient.On("BlockByHash", mock.Anything, newBlocks[i].Hash()).Return(newBlocks[i], nil).Once()
			}

			prevTd := int64(14)

			// insert new Tds for blocks 15 ~ 18; each block has new Td of 5
			for i := int64(15); i <= 18; i++ {
				td := prevTd + 5*(i-14)
				parent := newBlocks[i].ParentHash().Bytes()
				mockStoreManager.On("FindTd", mock.Anything, parent).Return(&model.TotalDifficulty{
					i - 1, parent, strconv.Itoa(int(td - 5))}, nil).Once()
				mockStoreManager.On("InsertTd", mock.Anything, idxCommon.TotalDifficulty(newBlocks[i], big.NewInt(td))).Return(nil).Once()
			}

			mockStoreManager.On("FindBlockByNumber", mock.Anything, int64(14)).Return(&model.Header{
				Number: 14,
				Hash:   blocks[14].Hash().Bytes(),
			}, nil).Once()

			// state diff for the new blocks
			for i := int64(15); i <= 18; i++ {
				mockEthClient.On("GetBlockReceipts", mock.Anything, newBlocks[i].Hash()).Return(types.Receipts{receipt}, nil).Once()
				mockEthClient.On("GetTransferLogs", mock.Anything, newBlocks[i].Hash()).Return(nil, nil).Once()
			}
			mockStoreManager.On("UpdateBlocks", mock.Anything, mockEthClient, newBlocks[15:19], [][]*types.Receipt{{receipt}, {receipt}, {receipt}, {receipt}}, [][]*types.TransferLog{nilTransferLogs, nilTransferLogs, nilTransferLogs, nilTransferLogs}, &model.Reorg{
				From:     15,
				FromHash: blocks[15].Hash().Bytes(),
				To:       15,
				ToHash:   blocks[15].Hash().Bytes(),
			}).Return(nil).Once()

			mockSubscriber.On("SubscribeNewHead", mock.Anything, mock.Anything).Return(subFunc, nil).Once()

			go func() {
				time.Sleep(time.Second)
				ch <- &multiclient.Header{
					Header: newBlocks[18].Header(),
				}
				time.Sleep(time.Second)
				cancel()
			}()

			err := idx.Listen(ctx, 0)
			Expect(err).Should(Equal(context.Canceled))
		})
	})

	Context("Listen() reorg with older header", func() {
		ctx, cancel := context.WithCancel(ctx)

		It("works fine", func() {
			// Given local state has blocks 10 ~ 17,
			// received the header 16, 17 from header channel, ignore it because current block number is larger
			// received the header 18 from header channel, and found the chain was reorg'ed at block 15

			// set up old blocks: 10 ~ 17
			blocks := make([]*types.Block, 20)
			tx := types.NewTransaction(0, common.Address{}, common.Big0, 0, common.Big0, []byte{})
			receipt := types.NewReceipt([]byte{}, false, 0)
			parentHash := common.HexToHash("123456789012345678901234567890")
			for i := int64(10); i <= 17; i++ {
				blocks[i] = types.NewBlock(
					&types.Header{
						Number:     big.NewInt(i),
						ParentHash: parentHash,
						Root:       common.HexToHash("1234567890" + strconv.Itoa(int(i))),
						Difficulty: big.NewInt(1),
					}, []*types.Transaction{tx}, nil, []*types.Receipt{receipt})
				parentHash = blocks[i].Hash()
			}

			// set up new blocks: 15 ~ 18
			newBlocks := make([]*types.Block, 20)
			copy(newBlocks, blocks)
			newTx := types.NewTransaction(0, common.Address{19, 23}, common.Big0, 0, common.Big0, []byte{19, 23})
			parentHash = blocks[14].Hash()
			for i := int64(15); i <= 18; i++ {
				newBlocks[i] = types.NewBlock(
					&types.Header{
						Number:     big.NewInt(i),
						ParentHash: parentHash,
						Root:       common.HexToHash("1234567890" + strconv.Itoa(int(i))),
						Difficulty: big.NewInt(5),
					}, []*types.Transaction{newTx}, nil, []*types.Receipt{receipt})
				parentHash = newBlocks[i].Hash()
			}

			// func getLocalState()
			// set init state when call Listen()
			mockStoreManager.On("FindLatestBlock", mock.Anything).Return(&model.Header{
				Number: 17,
				Hash:   blocks[17].Hash().Bytes(),
			}, nil).Once()
			mockStoreManager.On("FindTd", mock.Anything, blocks[17].Hash().Bytes()).Return(&model.TotalDifficulty{
				10, blocks[17].Hash().Bytes(), strconv.Itoa(17)}, nil).Once()

			// calculating Td for the new blocks 15 ~ 16
			mockEthClient.On("GetTotalDifficulty", mock.Anything, newBlocks[18].Hash()).Return(big.NewInt(34), nil).Once()

			// insert new TDs for 15 and 16, each block has TD of 5

			// insert new Tds for blocks 15 and 16, each block has Td of 5
			for i := int64(15); i <= 18; i++ {
				mockEthClient.On("BlockByHash", mock.Anything, newBlocks[i].Hash()).Return(newBlocks[i], nil).Once()
				td := 14 + 5*(i-14)
				parent := newBlocks[i].ParentHash().Bytes()
				mockStoreManager.On("FindTd", mock.Anything, parent).Return(&model.TotalDifficulty{
					i - 1, parent, strconv.Itoa(int(td - 5))}, nil).Once()
				mockStoreManager.On("InsertTd", mock.Anything, idxCommon.TotalDifficulty(newBlocks[i], big.NewInt(td))).Return(nil).Once()
			}

			// during reorg tracing, we query local db headers for headers to find the common ancestor of the new and old chain
			for i := int64(14); i <= 16; i++ {
				mockStoreManager.On("FindBlockByNumber", mock.Anything, i).Return(&model.Header{
					Number: i,
					Hash:   blocks[i].Hash().Bytes(),
				}, nil).Once()
			}

			// state diff for the new blocks
			for i := int64(15); i <= 18; i++ {
				mockEthClient.On("GetBlockReceipts", mock.Anything, newBlocks[i].Hash()).Return(types.Receipts{receipt}, nil).Once()
				mockEthClient.On("GetTransferLogs", mock.Anything, newBlocks[i].Hash()).Return(nil, nil).Once()
			}
			mockStoreManager.On("UpdateBlocks", mock.Anything, mockEthClient, newBlocks[15:19], [][]*types.Receipt{{receipt}, {receipt}, {receipt}, {receipt}}, [][]*types.TransferLog{nilTransferLogs, nilTransferLogs, nilTransferLogs, nilTransferLogs}, &model.Reorg{
				From:     15,
				FromHash: blocks[15].Hash().Bytes(),
				To:       17,
				ToHash:   blocks[17].Hash().Bytes(),
			}).Return(nil).Once()

			mockSubscriber.On("SubscribeNewHead", mock.Anything, mock.Anything).Return(subFunc, nil).Once()

			go func() {
				time.Sleep(time.Second)
				ch <- &multiclient.Header{
					Header: newBlocks[16].Header(),
				}
				ch <- &multiclient.Header{
					Header: newBlocks[17].Header(),
				}
				ch <- &multiclient.Header{
					Header: newBlocks[18].Header(),
				}
				time.Sleep(time.Second)
				cancel()
			}()

			err := idx.Listen(ctx, 0)
			Expect(err).Should(Equal(context.Canceled))
		})
	})
})

func TestIndexer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Indexer Test")
}
