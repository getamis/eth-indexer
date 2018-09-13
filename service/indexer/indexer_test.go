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
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/getamis/eth-indexer/client"
	clientMocks "github.com/getamis/eth-indexer/client/mocks"
	"github.com/getamis/eth-indexer/model"
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
		mockEthClients   []client.EthClient
		mockStoreManager *storeMocks.Manager
		idx              *indexer
		nilTransferLogs  []*types.TransferLog
		nilReorg         *model.Reorg
	)
	BeforeEach(func() {
		mockSub = &testSub{make(chan error)}
		mockStoreManager = new(storeMocks.Manager)
		mockEthClient = new(clientMocks.EthClient)

		// make mockEthClient the idx.latestClient
		mockEthClients = []client.EthClient{mockEthClient}
		idx = New(mockEthClients, mockStoreManager)
	})

	AfterEach(func() {
		mockStoreManager.AssertExpectations(GinkgoT())
		mockEthClient.AssertExpectations(GinkgoT())
	})

	Context("SubscribeErc20Tokens()", func() {
		ctx := context.Background()

		It("with valid parameters", func() {
			addresses := []string{"0x1234567890123456789012345678901234567890", "0x1234567890123456789012345678901234567891"}
			ethAddresses := []common.Address{common.HexToAddress(addresses[0]), common.HexToAddress(addresses[1])}
			mockStoreManager.On("Init", idx.latestClient).Return(nil).Once()
			// The first erc20 is not found
			mockStoreManager.On("FindERC20", ethAddresses[0]).Return(nil, gorm.ErrRecordNotFound).Once()
			erc20 := &model.ERC20{
				Address:     ethAddresses[0].Bytes(),
				BlockNumber: 0,
				Name:        "name",
				Decimals:    18,
				TotalSupply: "123",
			}
			mockEthClient.On("GetERC20", ctx, ethAddresses[0]).Return(erc20, nil).Once()
			mockStoreManager.On("InsertERC20", erc20).Return(nil).Once()
			// The second erc20 exists
			mockStoreManager.On("FindERC20", ethAddresses[1]).Return(nil, nil).Once()
			err := idx.SubscribeErc20Tokens(ctx, addresses)
			Expect(err).Should(BeNil())
		})

		Context("with invalid parameters", func() {
			unknownErr := errors.New("unknown error")
			It("failed to init store manager", func() {
				addresses := []string{"0x1234567890123456789012345678901234567890", "0x1234567890123456789012345678901234567891"}
				ethAddresses := []common.Address{common.HexToAddress(addresses[0]), common.HexToAddress(addresses[1])}
				mockStoreManager.On("Init", idx.latestClient).Return(unknownErr).Once()
				// The first erc20 is not found
				mockStoreManager.On("FindERC20", ethAddresses[0]).Return(nil, gorm.ErrRecordNotFound).Once()
				erc20 := &model.ERC20{
					Address:     ethAddresses[0].Bytes(),
					BlockNumber: 0,
					Name:        "name",
					Decimals:    18,
					TotalSupply: "123",
				}
				mockEthClient.On("GetERC20", ctx, ethAddresses[0]).Return(erc20, nil).Once()
				mockStoreManager.On("InsertERC20", erc20).Return(nil).Once()
				// The second erc20 exists
				mockStoreManager.On("FindERC20", ethAddresses[1]).Return(nil, nil).Once()
				err := idx.SubscribeErc20Tokens(ctx, addresses)
				Expect(err).Should(Equal(unknownErr))
			})

			It("failed to insert ERC20", func() {
				addresses := []string{"0x1234567890123456789012345678901234567890"}
				ethAddresses := []common.Address{common.HexToAddress(addresses[0])}
				// The first erc20 is not found
				mockStoreManager.On("FindERC20", ethAddresses[0]).Return(nil, gorm.ErrRecordNotFound).Once()
				erc20 := &model.ERC20{
					Address:     ethAddresses[0].Bytes(),
					BlockNumber: 0,
					Name:        "name",
					Decimals:    18,
					TotalSupply: "123",
				}
				mockEthClient.On("GetERC20", ctx, ethAddresses[0]).Return(erc20, nil).Once()
				mockStoreManager.On("InsertERC20", erc20).Return(unknownErr).Once()
				err := idx.SubscribeErc20Tokens(ctx, addresses)
				Expect(err).Should(Equal(unknownErr))
			})

			It("failed to get ERC20 from client", func() {
				addresses := []string{"0x1234567890123456789012345678901234567890"}
				ethAddresses := []common.Address{common.HexToAddress(addresses[0])}
				// The first erc20 is not found
				mockStoreManager.On("FindERC20", ethAddresses[0]).Return(nil, gorm.ErrRecordNotFound).Once()
				mockEthClient.On("GetERC20", ctx, ethAddresses[0]).Return(nil, unknownErr).Once()
				err := idx.SubscribeErc20Tokens(ctx, addresses)
				Expect(err).Should(Equal(unknownErr))
			})

			It("failed to find ERC20", func() {
				addresses := []string{"0x1234567890123456789012345678901234567890"}
				ethAddresses := []common.Address{common.HexToAddress(addresses[0])}
				// The first erc20 is not found
				mockStoreManager.On("FindERC20", ethAddresses[0]).Return(nil, unknownErr).Once()
				err := idx.SubscribeErc20Tokens(ctx, addresses)
				Expect(err).Should(Equal(unknownErr))
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

	Context("Listen()", func() {
		ch := make(chan *types.Header)
		unknownErr := errors.New("unknown error")

		Context("it works fine", func() {
			It("insert blocks in sequential", func() {
				// Given local state has the block 10,
				// receive new 18 & 19 blocks from header channel

				ctx, cancel := context.WithCancel(context.Background())
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

					mockEthClient.On("BlockByNumber", mock.Anything, big.NewInt(i)).Return(block, nil).Once()
					mockStoreManager.On("GetTd", block.Hash().Bytes()).Return(nil, gorm.ErrRecordNotFound).Once()
					mockEthClient.On("GetTotalDifficulty", mock.Anything, block.Hash()).Return(nil, gorm.ErrRecordNotFound).Once()

					parent := block.ParentHash().Bytes()
					mockStoreManager.On("GetTd", parent).Return(&model.TotalDifficulty{
						i - i, parent, strconv.Itoa(int(i - 1))}, nil).Once()
					mockStoreManager.On("InsertTd", block, big.NewInt(i)).Return(nil).Once()
					mockEthClient.On("GetBlockReceipts", mock.Anything, blocks[i].Hash()).Return(types.Receipts{receipt}, nil).Once()
					mockEthClient.On("GetTransferLogs", mock.Anything, blocks[i].Hash()).Return(nil, nil).Once()
					mockStoreManager.On("UpdateBlocks", mock.Anything, []*types.Block{block}, [][]*types.Receipt{{receipt}}, [][]*types.TransferLog{nilTransferLogs}, nilReorg).Return(nil).Once()
				}

				// deal with the new header 18,
				// blocks from 11 to 18
				// func getLocalState()
				mockStoreManager.On("LatestHeader").Return(&model.Header{
					Number: 10,
					Hash:   blocks[10].Hash().Bytes(),
				}, nil).Once()
				mockStoreManager.On("GetTd", blocks[10].Hash().Bytes()).Return(&model.TotalDifficulty{
					10, blocks[10].Hash().Bytes(), strconv.Itoa(10)}, nil).Once()

				// deal with the new header 19,
				// blocks from 18 to 19
				// func getLocalState()
				mockStoreManager.On("LatestHeader").Return(&model.Header{
					Number: 18,
					Hash:   blocks[18].Hash().Bytes(),
				}, nil).Once()
				mockStoreManager.On("GetTd", blocks[18].Hash().Bytes()).Return(&model.TotalDifficulty{
					18, blocks[18].Hash().Bytes(), strconv.Itoa(18)}, nil).Once()

				var recvCh chan<- *types.Header
				recvCh = ch
				mockEthClient.On("SubscribeNewHead", mock.Anything, recvCh).Return(mockSub, nil).Times(len(idx.clients))

				go func() {
					ch <- blocks[18].Header()
					ch <- blocks[19].Header()
					time.Sleep(time.Second)
					cancel()
				}()

				err := idx.Listen(ctx, ch, 0)
				Expect(err).Should(Equal(context.Canceled))
			})

			It("start indexer with a given block and cause block data gap in database", func() {
				// Given local state has the block 10, and start indexer with the block 15.
				// Receive 18, 19 blocks from header channel.

				ctx, cancel := context.WithCancel(context.Background())
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

					// record blocks from 15 to 19 in database
					if i >= 15 {
						mockEthClient.On("BlockByNumber", mock.Anything, big.NewInt(i)).Return(block, nil).Once()
						mockStoreManager.On("GetTd", block.Hash().Bytes()).Return(nil, gorm.ErrRecordNotFound).Once()
						mockEthClient.On("GetTotalDifficulty", mock.Anything, block.Hash()).Return(nil, gorm.ErrRecordNotFound).Once()

						parent := block.ParentHash().Bytes()
						mockStoreManager.On("GetTd", parent).Return(&model.TotalDifficulty{
							i - i, parent, strconv.Itoa(int(i - 1))}, nil).Once()
						mockStoreManager.On("InsertTd", block, big.NewInt(i)).Return(nil).Once()
						mockEthClient.On("GetBlockReceipts", mock.Anything, blocks[i].Hash()).Return(types.Receipts{receipt}, nil).Once()
						mockEthClient.On("GetTransferLogs", mock.Anything, blocks[i].Hash()).Return(nil, nil).Once()
						mockStoreManager.On("UpdateBlocks", mock.Anything, []*types.Block{block}, [][]*types.Receipt{{receipt}}, [][]*types.TransferLog{nilTransferLogs}, nilReorg).Return(nil).Once()
					}
				}

				// deal with the start header 15,
				// blocks from 15 to 18
				// func getLocalState()
				mockStoreManager.On("LatestHeader").Return(&model.Header{
					Number: 10,
					Hash:   blocks[10].Hash().Bytes(),
				}, nil).Once()
				mockEthClient.On("BlockByNumber", mock.Anything, big.NewInt(14)).Return(blocks[14], nil).Once()
				mockStoreManager.On("GetTd", blocks[14].Hash().Bytes()).Return(&model.TotalDifficulty{
					14, blocks[14].Hash().Bytes(), strconv.Itoa(14)}, nil).Once()

				// deal with the new header 19,
				// blocks from 18 to 19
				// func getLocalState()
				mockStoreManager.On("LatestHeader").Return(&model.Header{
					Number: 18,
					Hash:   blocks[18].Hash().Bytes(),
				}, nil).Once()
				mockStoreManager.On("GetTd", blocks[18].Hash().Bytes()).Return(&model.TotalDifficulty{
					18, blocks[18].Hash().Bytes(), strconv.Itoa(18)}, nil).Once()

				var recvCh chan<- *types.Header
				recvCh = ch
				mockEthClient.On("SubscribeNewHead", mock.Anything, recvCh).Return(mockSub, nil).Times(len(idx.clients))

				go func() {
					ch <- blocks[18].Header()
					ch <- blocks[19].Header()
					time.Sleep(time.Second)
					cancel()
				}()

				err := idx.Listen(ctx, ch, 15)
				Expect(err).Should(Equal(context.Canceled))
			})

			It("insert blocks with empty database", func() {
				// Given a empty database and a new header 19.
				// Should insert all the new blocks 0 ~ 19.

				ctx, cancel := context.WithCancel(context.Background())
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

				// func getLocalState()
				mockStoreManager.On("LatestHeader").Return(nil, gorm.ErrRecordNotFound).Once()

				// func addBlockMaybeReorg()
				mockEthClient.On("BlockByNumber", mock.Anything, big.NewInt(0)).Return(block, nil).Once()
				mockStoreManager.On("GetTd", block.Hash().Bytes()).Return(nil, gorm.ErrRecordNotFound).Once()
				mockEthClient.On("GetTotalDifficulty", mock.Anything, block.Hash()).Return(nil, gorm.ErrRecordNotFound).Once()

				mockStoreManager.On("InsertTd", block, big.NewInt(1)).Return(nil).Once()
				mockStoreManager.On("UpdateBlocks", mock.Anything, []*types.Block{block}, [][]*types.Receipt{{}}, [][]*types.TransferLog{{}}, nilReorg).Return(nil).Once()

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

					mockEthClient.On("BlockByNumber", mock.Anything, big.NewInt(i)).Return(block, nil).Once()
					mockStoreManager.On("GetTd", block.Hash().Bytes()).Return(nil, gorm.ErrRecordNotFound).Once()
					mockEthClient.On("GetTotalDifficulty", mock.Anything, block.Hash()).Return(nil, gorm.ErrRecordNotFound).Once()

					parent := block.ParentHash().Bytes()
					mockStoreManager.On("GetTd", parent).Return(&model.TotalDifficulty{
						i - i, parent, strconv.Itoa(int(i))}, nil).Once()

					// func insertBlocks()
					mockStoreManager.On("InsertTd", block, big.NewInt(i+1)).Return(nil).Once()
					mockEthClient.On("GetBlockReceipts", mock.Anything, blocks[i].Hash()).Return(types.Receipts{receipt}, nil).Once()
					mockEthClient.On("GetTransferLogs", mock.Anything, blocks[i].Hash()).Return(nil, nil).Once()
					mockStoreManager.On("UpdateBlocks", mock.Anything, []*types.Block{block}, [][]*types.Receipt{{receipt}}, [][]*types.TransferLog{nilTransferLogs}, nilReorg).Return(nil).Once()
				}

				var recvCh chan<- *types.Header
				recvCh = ch
				mockEthClient.On("SubscribeNewHead", mock.Anything, recvCh).Return(mockSub, nil).Times(len(idx.clients))

				go func() {
					ch <- blocks[19].Header()
					time.Sleep(time.Second)
					cancel()
				}()

				err := idx.Listen(ctx, ch, 0)
				Expect(err).Should(Equal(context.Canceled))
			})

			It("ignore old block", func() {
				ctx, cancel := context.WithCancel(context.Background())
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

				var recvCh chan<- *types.Header
				recvCh = ch
				mockEthClient.On("SubscribeNewHead", mock.Anything, recvCh).Return(mockSub, nil).Once()

				go func() {
					// Ignore old block
					ch <- blocks[10].Header()
					// Ignore old block
					ch <- blocks[14].Header()
					time.Sleep(time.Second)
					cancel()
				}()

				err := idx.Listen(ctx, ch, 15)
				Expect(err).Should(Equal(context.Canceled))
			})

			It("ignore the blocks which have already been recorded in database", func() {
				// Given local state has the block 10.
				// Receive block 15 from header channel first, insert blocks 11 ~ 15.
				// Receive block 13 from header channel then ignore directly.

				ctx, cancel := context.WithCancel(context.Background())
				blocks := make([]*types.Block, 20)

				tx := types.NewTransaction(0, common.Address{}, common.Big0, 0, common.Big0, []byte{})
				receipt := types.NewReceipt([]byte{}, false, 0)

				block := types.NewBlock(
					&types.Header{
						Number: big.NewInt(10),
						Root:   common.HexToHash("1234567890" + strconv.Itoa(int(10))),
					}, []*types.Transaction{tx}, nil, []*types.Receipt{receipt})
				blocks[10] = block

				// Receive block 15 from header channel first, insert blocks 11 ~ 15.
				// func getLocalState()
				mockStoreManager.On("LatestHeader").Return(&model.Header{
					Number: 10,
					Hash:   blocks[10].Hash().Bytes(),
				}, nil).Once()
				mockStoreManager.On("GetTd", blocks[10].Hash().Bytes()).Return(&model.TotalDifficulty{
					10, block.Hash().Bytes(), strconv.Itoa(10)}, nil).Once()

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

					mockEthClient.On("BlockByNumber", mock.Anything, big.NewInt(i)).Return(block, nil).Once()
					mockStoreManager.On("GetTd", block.Hash().Bytes()).Return(nil, gorm.ErrRecordNotFound).Once()
					mockEthClient.On("GetTotalDifficulty", mock.Anything, block.Hash()).Return(nil, gorm.ErrRecordNotFound).Once()

					parent := block.ParentHash().Bytes()
					mockStoreManager.On("GetTd", parent).Return(&model.TotalDifficulty{
						i - i, parent, strconv.Itoa(int(i - 1))}, nil).Once()

					// func insertBlocks()
					mockStoreManager.On("InsertTd", block, big.NewInt(i)).Return(nil).Once()
					mockEthClient.On("GetBlockReceipts", mock.Anything, blocks[i].Hash()).Return(types.Receipts{receipt}, nil).Once()
					mockEthClient.On("GetTransferLogs", mock.Anything, blocks[i].Hash()).Return(nil, nil).Once()
					mockStoreManager.On("UpdateBlocks", mock.Anything, []*types.Block{block}, [][]*types.Receipt{{receipt}}, [][]*types.TransferLog{nilTransferLogs}, nilReorg).Return(nil).Once()
				}

				// Receive block 13 from header channel then ignore directly.
				// func getLocalState()
				mockStoreManager.On("LatestHeader").Return(&model.Header{
					Number: 15,
					Hash:   blocks[15].Hash().Bytes(),
				}, nil).Once()
				mockStoreManager.On("GetTd", blocks[15].Hash().Bytes()).Return(&model.TotalDifficulty{
					15, blocks[15].Hash().Bytes(), strconv.Itoa(15)}, nil).Once()

				// func addBlockMaybeReorg()
				mockEthClient.On("BlockByNumber", mock.Anything, big.NewInt(13)).Return(blocks[13], nil).Once()
				mockStoreManager.On("GetTd", blocks[13].Hash().Bytes()).Return(&model.TotalDifficulty{
					13, blocks[13].Hash().Bytes(), strconv.Itoa(13)}, nil).Once()

				var recvCh chan<- *types.Header
				recvCh = ch
				mockEthClient.On("SubscribeNewHead", mock.Anything, recvCh).Return(mockSub, nil).Times(len(idx.clients))

				go func() {
					ch <- blocks[15].Header()
					ch <- blocks[13].Header()
					time.Sleep(time.Second)
					cancel()
				}()

				err := idx.Listen(ctx, ch, 0)
				Expect(err).Should(Equal(context.Canceled))
			})
		})

		Context("something goes wrong", func() {
			It("failed to GetTd()", func() {
				// Given init state has the block 10 but failed to get its total difficulty.

				ctx, cancel := context.WithCancel(context.Background())
				tx := types.NewTransaction(0, common.Address{}, common.Big0, 0, common.Big0, []byte{})
				receipt := types.NewReceipt([]byte{}, false, 0)
				block := types.NewBlock(
					&types.Header{
						Number: big.NewInt(10),
						Root:   common.HexToHash("1234567890" + strconv.Itoa(int(10))),
					}, []*types.Transaction{tx}, nil, []*types.Receipt{receipt})

				// func getLocalState()
				mockStoreManager.On("LatestHeader").Return(&model.Header{
					Number: 10,
					Hash:   block.Hash().Bytes(),
				}, nil).Once()

				// cause error here
				mockStoreManager.On("GetTd", block.Hash().Bytes()).Return(&model.TotalDifficulty{}, unknownErr).Once()

				var recvCh chan<- *types.Header
				recvCh = ch
				mockEthClient.On("SubscribeNewHead", mock.Anything, recvCh).Return(mockSub, nil).Times(len(idx.clients))

				go func() {
					ch <- block.Header()
					time.Sleep(time.Second)
					cancel()
				}()

				err := idx.Listen(ctx, ch, 0)
				Expect(err).Should(Equal(unknownErr))
			})

			It("failed to InsertTd()", func() {
				// Given init state has the block 10.
				// Received new header 11 but failed to insert total difficulty.

				ctx, cancel := context.WithCancel(context.Background())
				tx := types.NewTransaction(0, common.Address{}, common.Big0, 0, common.Big0, []byte{})
				receipt := types.NewReceipt([]byte{}, false, 0)
				block := types.NewBlock(
					&types.Header{
						Number: big.NewInt(10),
						Root:   common.HexToHash("1234567890" + strconv.Itoa(int(10))),
					}, []*types.Transaction{tx}, nil, []*types.Receipt{receipt})

				// func getLocalState()
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

				// insert the new block 11
				// func addBlockMaybeReorg()
				mockEthClient.On("BlockByNumber", mock.Anything, big.NewInt(11)).Return(block, nil).Once()
				mockStoreManager.On("GetTd", block.Hash().Bytes()).Return(nil, gorm.ErrRecordNotFound).Once()
				mockEthClient.On("GetTotalDifficulty", mock.Anything, block.Hash()).Return(nil, gorm.ErrRecordNotFound).Once()
				parent := block.ParentHash().Bytes()
				mockStoreManager.On("GetTd", parent).Return(&model.TotalDifficulty{
					10, parent, strconv.Itoa(10)}, nil).Once()

				// cause error here
				mockStoreManager.On("InsertTd", block, big.NewInt(11)).Return(unknownErr).Once()

				var recvCh chan<- *types.Header
				recvCh = ch
				mockEthClient.On("SubscribeNewHead", mock.Anything, recvCh).Return(mockSub, nil).Times(len(idx.clients))

				go func() {
					// new header: 11
					ch <- block.Header()
					time.Sleep(time.Second)
					cancel()
				}()

				err := idx.Listen(ctx, ch, 0)
				Expect(err).Should(Equal(unknownErr))
			})

			It("failed to insert block to database via UpdateBlocks()", func() {
				// Given init state has the block 10.
				// Received new header 11 but failed to update the block 11.

				ctx, cancel := context.WithCancel(context.Background())
				tx := types.NewTransaction(0, common.Address{}, common.Big0, 0, common.Big0, []byte{})
				receipt := types.NewReceipt([]byte{}, false, 0)

				// the existed block 10 in database
				block := types.NewBlock(
					&types.Header{
						Number: big.NewInt(10),
						Root:   common.HexToHash("1234567890" + strconv.Itoa(int(10))),
					}, []*types.Transaction{tx}, nil, []*types.Receipt{receipt})

				// func getLocalState()
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

				// insert the new block 11
				// func addBlockMaybeReorg()
				mockEthClient.On("BlockByNumber", mock.Anything, big.NewInt(11)).Return(block, nil).Once()
				mockStoreManager.On("GetTd", block.Hash().Bytes()).Return(nil, gorm.ErrRecordNotFound).Once()
				mockEthClient.On("GetTotalDifficulty", mock.Anything, block.Hash()).Return(nil, gorm.ErrRecordNotFound).Once()

				parent := block.ParentHash().Bytes()
				mockStoreManager.On("GetTd", parent).Return(&model.TotalDifficulty{
					10, parent, strconv.Itoa(10)}, nil).Once()
				mockStoreManager.On("InsertTd", block, big.NewInt(11)).Return(nil).Once()
				mockEthClient.On("GetBlockReceipts", mock.Anything, block.Hash()).Return(types.Receipts{receipt}, nil).Once()
				mockEthClient.On("GetTransferLogs", mock.Anything, block.Hash()).Return(nil, nil).Once()

				// cause error here
				mockStoreManager.On("UpdateBlocks", mock.Anything, []*types.Block{block}, [][]*types.Receipt{{receipt}}, [][]*types.TransferLog{nilTransferLogs}, nilReorg).Return(unknownErr).Once()

				var recvCh chan<- *types.Header
				recvCh = ch
				mockEthClient.On("SubscribeNewHead", mock.Anything, recvCh).Return(mockSub, nil).Times(len(idx.clients))

				go func() {
					// New heaser: block 11
					ch <- block.Header()
					time.Sleep(time.Second)
					cancel()
				}()

				err := idx.Listen(ctx, ch, 0)
				Expect(err).Should(Equal(unknownErr))
			})

			It("failed to get transaction receipt via GetBlockReceipts()", func() {
				// Given init state has the block 10.
				// Received new header 11 but failed to get the receipt of block 11.

				ctx, cancel := context.WithCancel(context.Background())
				tx := types.NewTransaction(0, common.Address{}, common.Big0, 0, common.Big0, []byte{})
				receipt := types.NewReceipt([]byte{}, false, 0)

				// the existed block 10 in database
				block := types.NewBlock(
					&types.Header{
						Number: big.NewInt(10),
						Root:   common.HexToHash("1234567890" + strconv.Itoa(int(10))),
					}, []*types.Transaction{tx}, nil, []*types.Receipt{receipt})

				// func getLocalState()
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

				// insert the new block 11
				// func addBlockMaybeReorg()
				mockEthClient.On("BlockByNumber", mock.Anything, big.NewInt(11)).Return(block, nil).Once()
				mockStoreManager.On("GetTd", block.Hash().Bytes()).Return(nil, gorm.ErrRecordNotFound).Once()
				mockEthClient.On("GetTotalDifficulty", mock.Anything, block.Hash()).Return(nil, gorm.ErrRecordNotFound).Once()
				parent := block.ParentHash().Bytes()
				mockStoreManager.On("GetTd", parent).Return(&model.TotalDifficulty{
					10, parent, strconv.Itoa(10)}, nil).Once()
				mockStoreManager.On("InsertTd", block, big.NewInt(11)).Return(nil).Once()

				// func getBlockData()
				// cause error here
				mockEthClient.On("GetBlockReceipts", mock.Anything, block.Hash()).Return(nil, unknownErr).Once()

				var recvCh chan<- *types.Header
				recvCh = ch
				mockEthClient.On("SubscribeNewHead", mock.Anything, recvCh).Return(mockSub, nil).Times(len(idx.clients))

				go func() {
					// new header 11
					ch <- block.Header()
					time.Sleep(time.Second)
					cancel()
				}()

				err := idx.Listen(ctx, ch, 0)
				Expect(err).Should(Equal(unknownErr))
			})

			It("failed to get block by number via BlockByNumber()", func() {
				// Given init state has the block 9.
				// Received new header 11 but failed to get the block info of 10.

				ctx, cancel := context.WithCancel(context.Background())
				tx := types.NewTransaction(0, common.Address{}, common.Big0, 0, common.Big0, []byte{})
				receipt := types.NewReceipt([]byte{}, false, 0)

				block := types.NewBlock(
					&types.Header{
						Number: big.NewInt(10),
						Root:   common.HexToHash("1234567890" + strconv.Itoa(int(10))),
					}, []*types.Transaction{tx}, nil, []*types.Receipt{receipt})

				// Given init state has the block 9
				// func getLocalState()
				mockStoreManager.On("LatestHeader").Return(&model.Header{
					Number: 9,
					Hash:   block.Hash().Bytes(),
				}, nil).Once()
				mockStoreManager.On("GetTd", block.Hash().Bytes()).Return(&model.TotalDifficulty{
					10, block.Hash().Bytes(), strconv.Itoa(9)}, nil).Once()

				// cause error here
				mockEthClient.On("BlockByNumber", mock.Anything, big.NewInt(10)).Return(nil, unknownErr).Once()

				var recvCh chan<- *types.Header
				recvCh = ch
				mockEthClient.On("SubscribeNewHead", mock.Anything, recvCh).Return(mockSub, nil).Times(len(idx.clients))

				go func() {
					ch <- block.Header()
					time.Sleep(time.Second)
					cancel()
				}()

				err := idx.Listen(ctx, ch, 0)
				Expect(err).Should(Equal(unknownErr))
			})

			It("failed to get latest header", func() {
				ctx, cancel := context.WithCancel(context.Background())

				var recvCh chan<- *types.Header
				recvCh = ch

				mockStoreManager.On("LatestHeader").Return(nil, unknownErr).Once()
				mockEthClient.On("SubscribeNewHead", mock.Anything, recvCh).Return(mockSub, nil).Times(len(idx.clients))

				go func() {
					ch <- &types.Header{
						Number: big.NewInt(10),
					}
					time.Sleep(time.Second)
					cancel()
				}()

				err := idx.Listen(ctx, ch, 0)
				Expect(err).Should(Equal(unknownErr))
			})

			Context("test connection with multiple ethClients", func() {
				var (
					mockSub          *testSub
					newMockSub       *testSub
					mockEthClient    *clientMocks.EthClient
					newMockEthClient *clientMocks.EthClient
					idx              *indexer
				)
				BeforeEach(func() {
					mockSub = &testSub{make(chan error)}
					newMockSub = &testSub{make(chan error)}
					mockEthClient = new(clientMocks.EthClient)
					newMockEthClient = new(clientMocks.EthClient)
					mockEthClients = []client.EthClient{newMockEthClient, mockEthClient}
					idx = New(mockEthClients, mockStoreManager)
				})

				It("failed to subscribe new event", func() {
					ctx := context.Background()
					var recvCh chan<- *types.Header
					recvCh = ch

					// make all the ethClients have this failure
					newMockEthClient.On("SubscribeNewHead", mock.Anything, recvCh).Return(nil, unknownErr).Once()
					mockEthClient.On("SubscribeNewHead", mock.Anything, recvCh).Return(nil, unknownErr).Once()

					err := idx.Listen(ctx, ch, 0)
					Expect(err).Should(Equal(unknownErr))
				})

				It("all ethClients return subscribe event errors", func() {
					ctx := context.Background()
					var recvCh chan<- *types.Header
					recvCh = ch

					// all the EthClients are gone
					// in this case, we will return error and restart indexer
					newMockEthClient.On("SubscribeNewHead", mock.Anything, recvCh).Return(newMockSub, unknownErr).Once()
					mockEthClient.On("SubscribeNewHead", mock.Anything, recvCh).Return(mockSub, unknownErr).Once()

					err := idx.Listen(ctx, ch, 0)
					Expect(err).Should(Equal(unknownErr))
				})

				It("not all ethClients return subscribe event errors", func() {
					ctx, cancel := context.WithCancel(context.Background())
					var recvCh chan<- *types.Header
					recvCh = ch

					// all the EthClients except the last one are gone
					// in this case, we will keep indexer alive
					newMockEthClient.On("SubscribeNewHead", mock.Anything, recvCh).Return(newMockSub, unknownErr).Once()
					// the latest ethClient works
					mockEthClient.On("SubscribeNewHead", mock.Anything, recvCh).Return(mockSub, nil).Once()

					// finish the test case by sending context cancel message
					go func() {
						time.Sleep(time.Second)
						cancel()
					}()

					err := idx.Listen(ctx, ch, 0)
					Expect(err).Should(Equal(context.Canceled))
				})

				It("all ethClients return subscribe head errors", func() {
					ctx := context.Background()
					subError := errors.New("client is closed")
					var recvCh chan<- *types.Header
					recvCh = ch

					// all the EthClients are gone
					// in this case, we will return error and restart indexer
					newMockEthClient.On("SubscribeNewHead", mock.Anything, recvCh).Return(newMockSub, nil).Once()
					mockEthClient.On("SubscribeNewHead", mock.Anything, recvCh).Return(mockSub, nil).Once()

					go func() {
						mockSub.mychan <- subError
						newMockSub.mychan <- subError
					}()

					err := idx.Listen(ctx, ch, 0)
					Expect(err).Should(Equal(subError))
				})

				It("not all ethClients return subscribe head errors", func() {
					ctx, cancel := context.WithCancel(context.Background())
					subError := errors.New("client is closed")
					var recvCh chan<- *types.Header
					recvCh = ch

					newMockEthClient.On("SubscribeNewHead", mock.Anything, recvCh).Return(newMockSub, nil).Once()
					mockEthClient.On("SubscribeNewHead", mock.Anything, recvCh).Return(mockSub, nil).Once()

					// one of the ethClient throwed subscription error
					go func() {
						newMockSub.mychan <- subError
						// finish the test case by sending context cancel message
						time.Sleep(time.Second)
						cancel()
					}()

					err := idx.Listen(ctx, ch, 0)
					Expect(err).Should(Equal(context.Canceled))
				})
			})
		})
	})

	Context("Listen() reorg the new blocks", func() {
		ctx, cancel := context.WithCancel(context.Background())
		ch := make(chan *types.Header)

		It("works fine", func() {
			// Given local state has the blocks 10 ~ 15,
			// received the new header 18 from header channel.
			// after inserting the blocks 16 ~ 17 and dealing with the block 18, we found that chain was reorg'ed at block 15 (blocks 15 ~ 18 were changed)

			// set up old blocks: 11 ~ 17
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
			// parentHash changed here
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
			// set expectations
			// when receiving the first header 18, checking the latest header in database
			mockStoreManager.On("LatestHeader").Return(&model.Header{
				Number: 15,
				Hash:   blocks[15].Hash().Bytes(),
			}, nil).Once()
			mockStoreManager.On("GetTd", blocks[15].Hash().Bytes()).Return(&model.TotalDifficulty{
				15, blocks[15].Hash().Bytes(), strconv.Itoa(15)}, nil).Once()

			// receiving the first header 18, syncing blocks from 16 to 18
			// notice that while dealing the block 18, we found the block is on different chain with different Difficulty and parentHash
			for i := int64(16); i <= 18; i++ {
				if i <= 17 {
					// so far reorg does not take place
					mockEthClient.On("BlockByNumber", mock.Anything, big.NewInt(i)).Return(blocks[i], nil).Once()
					mockStoreManager.On("GetTd", blocks[i].Hash().Bytes()).Return(nil, gorm.ErrRecordNotFound).Once()
					mockEthClient.On("GetTotalDifficulty", mock.Anything, blocks[i].Hash()).Return(nil, gorm.ErrRecordNotFound).Once()

					// insert old Tds for blocks 16 to 17, each block has TD of 1
					parent := blocks[i].ParentHash().Bytes()
					mockStoreManager.On("GetTd", parent).Return(&model.TotalDifficulty{
						i - 1, parent, strconv.Itoa(int(i - 1))}, nil).Once()
					mockStoreManager.On("InsertTd", blocks[i], big.NewInt(i)).Return(nil).Once()
				} else {
					// the block 18 indicates reorg is needed
					mockEthClient.On("BlockByNumber", mock.Anything, big.NewInt(i)).Return(newBlocks[i], nil).Once()
					mockStoreManager.On("GetTd", newBlocks[i].Hash().Bytes()).Return(nil, gorm.ErrRecordNotFound).Once()
					mockEthClient.On("GetTotalDifficulty", mock.Anything, newBlocks[i].Hash()).Return(nil, gorm.ErrRecordNotFound).Once()
				}
			}

			// during the reorg, we query block by hash to trace the canonical chain on ethereum from 17 to 15
			for i := int64(17); i >= 15; i-- {
				mockEthClient.On("BlockByHash", mock.Anything, newBlocks[i].Hash()).Return(newBlocks[i], nil).Once()
			}

			prevTd := int64(14)
			// calculating Td for the new blocks 15 ~ 16
			mockStoreManager.On("GetTd", blocks[14].Hash().Bytes()).Return(&model.TotalDifficulty{
				14, blocks[14].Hash().Bytes(), strconv.Itoa(14)}, nil).Once()

			// insert new Tds for blocks 15 ~ 18; each block has new Td of 5
			for i := int64(15); i <= 18; i++ {
				td := prevTd + 5*(i-14)
				parent := newBlocks[i].ParentHash().Bytes()
				mockStoreManager.On("GetTd", parent).Return(&model.TotalDifficulty{
					i - 1, parent, strconv.Itoa(int(td - 5))}, nil).Once()
				mockStoreManager.On("InsertTd", newBlocks[i], big.NewInt(td)).Return(nil).Once()
			}

			// during reorg tracing, we query local db headers to find the common ancestor of the new and old chain
			for i := int64(14); i <= 16; i++ {
				mockStoreManager.On("GetHeaderByNumber", i).Return(&model.Header{
					Number: i,
					Hash:   blocks[i].Hash().Bytes(),
				}, nil).Once()
			}

			// expectations for querying state diff
			// state diff for the old blocks
			for i := int64(16); i <= 17; i++ {
				mockEthClient.On("GetBlockReceipts", mock.Anything, blocks[i].Hash()).Return(types.Receipts{receipt}, nil).Once()
				mockEthClient.On("GetTransferLogs", mock.Anything, blocks[i].Hash()).Return(nil, nil).Once()
				mockStoreManager.On("UpdateBlocks", mock.Anything, []*types.Block{blocks[i]}, [][]*types.Receipt{{receipt}}, [][]*types.TransferLog{nilTransferLogs}, nilReorg).Return(nil).Once()
			}
			// state diff for the new blocks
			for i := int64(15); i <= 18; i++ {
				mockEthClient.On("GetBlockReceipts", mock.Anything, newBlocks[i].Hash()).Return(types.Receipts{receipt}, nil).Once()
				mockEthClient.On("GetTransferLogs", mock.Anything, newBlocks[i].Hash()).Return(nil, nil).Once()
			}
			mockStoreManager.On("UpdateBlocks", mock.Anything, newBlocks[15:19], [][]*types.Receipt{{receipt}, {receipt}, {receipt}, {receipt}}, [][]*types.TransferLog{nilTransferLogs, nilTransferLogs, nilTransferLogs, nilTransferLogs}, &model.Reorg{
				From:     15,
				FromHash: blocks[15].Hash().Bytes(),
				To:       17,
				ToHash:   blocks[17].Hash().Bytes(),
			}).Return(nil).Once()

			var recvCh chan<- *types.Header
			recvCh = ch
			mockEthClient.On("SubscribeNewHead", mock.Anything, recvCh).Return(mockSub, nil).Times(len(idx.clients))

			go func() {
				ch <- newBlocks[18].Header()
				time.Sleep(time.Second)
				cancel()
			}()

			err := idx.Listen(ctx, ch, 0)
			Expect(err).Should(Equal(context.Canceled))
		})
	})

	Context("Listen() reorg with older header", func() {
		ctx, cancel := context.WithCancel(context.Background())
		ch := make(chan *types.Header)

		It("works fine", func() {
			// Given local state has blocks 10 ~ 17,
			// received the header 16 from header channel and found the chain was reorg'ed at block 15

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

			// set up new blocks: 15 ~ 16
			newBlocks := make([]*types.Block, 20)
			copy(newBlocks, blocks)
			newTx := types.NewTransaction(0, common.Address{19, 23}, common.Big0, 0, common.Big0, []byte{19, 23})
			parentHash = blocks[14].Hash()
			for i := int64(15); i <= 16; i++ {
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
			// when receiving the first header 16, checking the latest header 17 in database
			mockStoreManager.On("LatestHeader").Return(&model.Header{
				Number: 17,
				Hash:   blocks[17].Hash().Bytes(),
			}, nil).Once()
			mockStoreManager.On("GetTd", blocks[17].Hash().Bytes()).Return(&model.TotalDifficulty{
				10, blocks[17].Hash().Bytes(), strconv.Itoa(17)}, nil).Once()

			// try to get new block info of 16 (new Td)
			// func addBlockMaybeReorg()
			mockEthClient.On("BlockByNumber", mock.Anything, big.NewInt(16)).Return(newBlocks[16], nil).Once()
			mockStoreManager.On("GetTd", newBlocks[16].Hash().Bytes()).Return(nil, gorm.ErrRecordNotFound).Once()
			mockEthClient.On("GetTotalDifficulty", mock.Anything, newBlocks[16].Hash()).Return(nil, gorm.ErrRecordNotFound).Once()

			// Reorg starts from here
			// during reorg tracing, we query local db headers for headers to find the common ancestor of the new and old chain
			for i := int64(14); i <= 15; i++ {
				mockStoreManager.On("GetHeaderByNumber", i).Return(&model.Header{
					Number: i,
					Hash:   blocks[i].Hash().Bytes(),
				}, nil).Once()
			}

			// calculating Td for the new blocks 15 ~ 16
			mockStoreManager.On("GetTd", blocks[14].Hash().Bytes()).Return(&model.TotalDifficulty{
				14, blocks[14].Hash().Bytes(), strconv.Itoa(14)}, nil).Once()

			// during reorg, we query block by hash to trace the canonical chain on ethereum from 16->14
			mockEthClient.On("BlockByHash", mock.Anything, newBlocks[15].Hash()).Return(newBlocks[15], nil).Once()

			// insert new Tds for blocks 15 and 16, each block has Td of 5
			for i := int64(15); i <= 16; i++ {
				td := 14 + 5*(i-14)
				parent := newBlocks[i].ParentHash().Bytes()
				mockStoreManager.On("GetTd", parent).Return(&model.TotalDifficulty{
					i - 1, parent, strconv.Itoa(int(td - 5))}, nil).Once()
				mockStoreManager.On("InsertTd", newBlocks[i], big.NewInt(td)).Return(nil).Once()
			}

			// state diff for the new blocks
			for i := int64(15); i <= 16; i++ {
				mockEthClient.On("GetBlockReceipts", mock.Anything, newBlocks[i].Hash()).Return(types.Receipts{receipt}, nil).Once()
				mockEthClient.On("GetTransferLogs", mock.Anything, newBlocks[i].Hash()).Return(nil, nil).Once()
			}
			mockStoreManager.On("UpdateBlocks", mock.Anything, newBlocks[15:17], [][]*types.Receipt{{receipt}, {receipt}}, [][]*types.TransferLog{nilTransferLogs, nilTransferLogs}, &model.Reorg{
				From:     15,
				FromHash: blocks[15].Hash().Bytes(),
				To:       17,
				ToHash:   blocks[17].Hash().Bytes(),
			}).Return(nil).Once()

			var recvCh chan<- *types.Header
			recvCh = ch
			mockEthClient.On("SubscribeNewHead", mock.Anything, recvCh).Return(mockSub, nil).Times(len(idx.clients))

			go func() {
				ch <- newBlocks[16].Header()
				time.Sleep(time.Second)
				cancel()
			}()

			err := idx.Listen(ctx, ch, 0)
			Expect(err).Should(Equal(context.Canceled))
		})
	})
})

func TestIndexer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Indexer Test")
}
