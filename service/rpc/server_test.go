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
package rpc

import (
	"context"
	"math/big"
	"strconv"
	"testing"

	"database/sql/driver"

	gethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/getamis/sirius/log"
	"github.com/jinzhu/gorm"
	"github.com/maichain/eth-indexer/common"
	"github.com/maichain/eth-indexer/model"
	. "github.com/maichain/eth-indexer/service"
	"github.com/maichain/eth-indexer/service/pb"
	"github.com/maichain/eth-indexer/store/mocks"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func makeHeader(number int64, hashHex string) *model.Header {
	return &model.Header{
		Hash:        common.HexToBytes(hashHex),
		ParentHash:  common.HexToBytes("0x35b9253b70be351059982e8d6a218146a18ef9b723e560c7efc540629b4e75f2"),
		UncleHash:   common.HexToBytes("0x2d6159f94932bd669c7161e2563ea4cc0fbf848dd59adbed7df3da74072edd50"),
		Coinbase:    common.HexToBytes("0xB287a379e6caCa6732E50b88D23c290aA990A892"),
		Root:        common.HexToBytes("0x86f9a7ccb763958d0f6c01ea89b7a49eb5a3a8aff0f998ff514b97ad1c4e1fd6"),
		TxHash:      common.HexToBytes("0x3f28c6504aa57084da641571cd710e092c716979dac2664f70fc62cd9d792a4b"),
		ReceiptHash: common.HexToBytes("0xad2ad2d0fca28f18d0d9fedc7ec2ab4b97277546c212f67519314bfb30f56736"),
		Difficulty:  927399944,
		Number:      number,
		GasLimit:    810000,
		GasUsed:     809999,
		Time:        123456789,
		MixDigest:   []byte{11, 23, 45},
		Nonce:       []byte{12, 13, 56, 77},
	}
}

func makeTx(txHex, blockHex string) *model.Transaction {
	return &model.Transaction{
		Hash:      common.HexToBytes(txHex),
		BlockHash: common.HexToBytes(blockHex),
		From:      common.HexToBytes("0xB287a379e6caCa6732E50b88D23c290aA990A892"),
		Nonce:     10013,
		GasPrice:  "123456789",
		GasLimit:  45000,
		Amount:    "4840283445",
		Payload:   []byte{12, 34},
	}
}

func makeBlockQueryResponse(header *model.Header, txs []*model.Transaction) *pb.BlockQueryResponse {
	response := &pb.BlockQueryResponse{
		Block: &pb.Block{
			Hash:   common.BytesTo0xHex(header.Hash),
			Number: header.Number,
			Nonce:  header.Nonce},
	}
	for _, transaction := range txs {
		tx := &pb.Transaction{
			Hash:     common.BytesTo0xHex(transaction.Hash),
			From:     common.BytesTo0xHex(transaction.From),
			Nonce:    transaction.Nonce,
			GasPrice: transaction.GasPrice,
			GasLimit: transaction.GasLimit,
			Amount:   transaction.Amount,
			Payload:  transaction.Payload,
		}
		if transaction.To != nil {
			tx.To = common.BytesTo0xHex(transaction.To)
		}
		response.Txs = append(response.Txs, tx)
	}
	return response
}

func makeTxQueryResponse(tx *model.Transaction) *pb.TransactionQueryResponse {
	return &pb.TransactionQueryResponse{
		Tx: &pb.Transaction{
			Hash:     common.BytesTo0xHex(tx.Hash),
			From:     common.BytesTo0xHex(tx.From),
			Nonce:    tx.Nonce,
			GasPrice: tx.GasPrice,
			GasLimit: tx.GasLimit,
			Amount:   tx.Amount,
			Payload:  tx.Payload,
		}}
}

var _ = Describe("Server Test", func() {
	var (
		mockServiceManager *mocks.ServiceManager
		svr                *server
	)

	BeforeEach(func() {
		mockServiceManager = new(mocks.ServiceManager)
		svr = New(mockServiceManager)
	})

	AfterEach(func() {
		mockServiceManager.AssertExpectations(GinkgoT())
	})

	Context("GetBlockByHash()", func() {
		ctx := context.Background()
		blockHashHex := "0x58bb59babd8fd8299b22acb997832a75d7b6b666579f80cc281764342f2b373b"
		req := &pb.BlockHashQueryRequest{Hash: blockHashHex}

		Context("block exists", func() {
			It("returns the block", func() {

				header := makeHeader(1000300, blockHashHex)
				mockServiceManager.On("FindBlockByHash", common.HexToBytes(blockHashHex)).Return(header, nil).Once()
				numTx := 10
				txs := make([]*model.Transaction, numTx)
				for i := 0; i < numTx; i++ {
					txs[i] = makeTx(common.StringToHex("transaction_"+strconv.Itoa(int(i))), blockHashHex)
				}
				mockServiceManager.On("FindTransactionsByBlockHash", common.HexToBytes(blockHashHex)).Return(txs, nil).Once()
				res, err := svr.GetBlockByHash(ctx, req)
				Expect(err).Should(Succeed())
				Expect(*res).Should(Equal(*makeBlockQueryResponse(header, txs)))
			})
		})

		Context("block does not exist", func() {
			It("returns error", func() {
				mockServiceManager.On("FindBlockByHash", common.HexToBytes(blockHashHex)).Return(nil, gorm.ErrRecordNotFound).Once()
				res, err := svr.GetBlockByHash(ctx, req)
				Expect(err).ShouldNot(BeNil())
				Expect(res).Should(BeNil())

				s, _ := status.FromError(err)
				Expect(s.Code()).Should(Equal(codes.NotFound))
			})
		})

		Context("transient error", func() {
			It("returns nothing", func() {
				mockServiceManager.On("FindBlockByHash", common.HexToBytes(blockHashHex)).Return(nil, driver.ErrBadConn).Once()
				res, err := svr.GetBlockByHash(ctx, req)
				Expect(err).ShouldNot(BeNil())
				Expect(res).Should(BeNil())

				s, _ := status.FromError(err)
				Expect(s.Code()).Should(Equal(codes.Internal))
			})

			It("returns whatever it has got", func() {
				header := makeHeader(1000300, blockHashHex)
				mockServiceManager.On("FindBlockByHash", common.HexToBytes(blockHashHex)).Return(header, nil).Once()
				mockServiceManager.On("FindTransactionsByBlockHash", common.HexToBytes(blockHashHex)).Return(nil, driver.ErrBadConn).Once()
				res, err := svr.GetBlockByHash(ctx, req)
				Expect(err).ShouldNot(BeNil())
				Expect(*res).Should(Equal(*makeBlockQueryResponse(header, []*model.Transaction{})))
			})
		})
	})

	Context("GetBlockByNumber()", func() {
		ctx := context.Background()
		blockNum := int64(1000300)
		req := &pb.BlockNumberQueryRequest{Number: blockNum}

		Context("block exists", func() {
			It("returns the block", func() {
				blockHashHex := "0x58bb59babd8fd8299b22acb997832a75d7b6b666579f80cc281764342f2b373b"
				header := makeHeader(blockNum, blockHashHex)
				mockServiceManager.On("FindBlockByNumber", blockNum).Return(header, nil).Once()
				numTx := 10
				txs := make([]*model.Transaction, numTx)
				for i := 0; i < numTx; i++ {
					txs[i] = makeTx(common.StringToHex("transaction_"+strconv.Itoa(int(i))), blockHashHex)
				}
				mockServiceManager.On("FindTransactionsByBlockHash", common.HexToBytes(blockHashHex)).Return(txs, nil).Once()
				res, err := svr.GetBlockByNumber(ctx, req)
				Expect(err).Should(Succeed())
				Expect(*res).Should(Equal(*makeBlockQueryResponse(header, txs)))
			})
		})

		Context("block does not exist", func() {
			It("returns error", func() {
				mockServiceManager.On("FindBlockByNumber", blockNum).Return(nil, gorm.ErrRecordNotFound).Once()
				res, err := svr.GetBlockByNumber(ctx, req)
				Expect(err).ShouldNot(BeNil())
				Expect(res).Should(BeNil())

				s, _ := status.FromError(err)
				Expect(s.Code()).Should(Equal(codes.NotFound))
			})
		})

		Context("transient error", func() {
			It("returns nothing", func() {
				mockServiceManager.On("FindBlockByNumber", blockNum).Return(nil, driver.ErrBadConn).Once()
				res, err := svr.GetBlockByNumber(ctx, req)
				Expect(err).ShouldNot(BeNil())
				Expect(res).Should(BeNil())

				s, _ := status.FromError(err)
				Expect(s.Code()).Should(Equal(codes.Internal))
			})

			It("returns whatever it has got", func() {
				blockHashHex := "0x58bb59babd8fd8299b22acb997832a75d7b6b666579f80cc281764342f2b373b"
				header := makeHeader(blockNum, blockHashHex)
				mockServiceManager.On("FindBlockByNumber", blockNum).Return(header, nil).Once()
				mockServiceManager.On("FindTransactionsByBlockHash", common.HexToBytes(blockHashHex)).Return(nil, driver.ErrBadConn).Once()
				res, err := svr.GetBlockByNumber(ctx, req)
				Expect(err).ShouldNot(BeNil())
				Expect(*res).Should(Equal(*makeBlockQueryResponse(header, []*model.Transaction{})))
			})
		})
	})

	Context("GetTransactionByHash()", func() {
		ctx := context.Background()
		txHashHex := "0x58bb59babd8fd8299b22acb997832a75d7b6b666579f80cc281764342f2b373b"
		req := &pb.TransactionQueryRequest{Hash: txHashHex}

		Context("tx exists", func() {
			It("returns the block", func() {
				tx := makeTx(txHashHex, "0x88bb59babd8fd8299b22acb997832a75d7b6b666579f80cc281764342f2b373b")
				mockServiceManager.On("FindTransaction", tx.Hash).Return(tx, nil).Once()
				res, err := svr.GetTransactionByHash(ctx, req)
				Expect(err).Should(Succeed())
				Expect(*res).Should(Equal(*makeTxQueryResponse(tx)))
			})
		})

		Context("tx does not exist", func() {
			It("returns empty response", func() {
				mockServiceManager.On("FindTransaction", common.HexToBytes(txHashHex)).Return(nil, gorm.ErrRecordNotFound).Once()
				res, err := svr.GetTransactionByHash(ctx, req)
				Expect(err).ShouldNot(BeNil())
				Expect(res).Should(BeNil())

				s, _ := status.FromError(err)
				Expect(s.Code()).Should(Equal(codes.NotFound))
			})
		})

		Context("transient error", func() {
			It("returns nothing", func() {
				mockServiceManager.On("FindTransaction", common.HexToBytes(txHashHex)).Return(nil, driver.ErrBadConn).Once()
				res, err := svr.GetTransactionByHash(ctx, req)
				Expect(err).ShouldNot(BeNil())
				Expect(res).Should(BeNil())

				s, _ := status.FromError(err)
				Expect(s.Code()).Should(Equal(codes.Internal))
			})
		})
	})
})

var _ = Describe("Server Balance Test", func() {
	var (
		mockServiceManager *mocks.ServiceManager
		svr                *server
	)

	BeforeEach(func() {
		mockServiceManager = new(mocks.ServiceManager)
		svr = &server{
			manager: mockServiceManager,
			logger:  log.Discard(),
		}
	})

	AfterEach(func() {
		mockServiceManager.AssertExpectations(GinkgoT())
	})

	Context("GetBalance()", func() {
		ctx := context.Background()

		Context("ETH", func() {
			blockNum := int64(5430100)
			req := &pb.GetBalanceRequest{
				Token:       ethToken,
				Address:     "0x343c43a37d37dff08ae8c4a11544c718abb4fcf8",
				BlockNumber: blockNum + 10}

			Context("account exists", func() {
				It("returns the balance", func() {
					balanceString := "987654321098765432109876543210"
					balance, ok := new(big.Int).SetString(balanceString, 10)
					Expect(ok).Should(BeTrue())
					mockServiceManager.On("GetBalance", ctx, gethCommon.HexToAddress(req.Address), req.BlockNumber).Return(balance, new(big.Int).SetInt64(blockNum), nil).Once()
					res, err := svr.GetBalance(ctx, req)
					Expect(err).Should(Succeed())
					expRes := &pb.GetBalanceResponse{Amount: balanceString, BlockNumber: blockNum}
					Expect(res).Should(Equal(expRes))
				})
			})

			Context("account does not exist", func() {
				It("returns error", func() {
					mockServiceManager.On("GetBalance", ctx, gethCommon.HexToAddress(req.Address), req.BlockNumber).Return(nil, nil, gorm.ErrRecordNotFound).Once()
					res, err := svr.GetBalance(ctx, req)
					Expect(err).ShouldNot(Equal(gorm.ErrRecordNotFound))
					Expect(res).Should(BeNil())
				})
			})

			Context("transient error", func() {
				It("returns error", func() {
					mockServiceManager.On("GetBalance", ctx, gethCommon.HexToAddress(req.Address), req.BlockNumber).Return(nil, nil, driver.ErrBadConn).Once()
					res, err := svr.GetBalance(ctx, req)
					Expect(err).ShouldNot(Equal(driver.ErrBadConn))
					Expect(res).Should(BeNil())
				})
			})
		})

		Context("ERC20", func() {
			blockNum := int64(5430100)
			req := &pb.GetBalanceRequest{
				Token:       "0xfffd933a0bc612844eaf0c6fe3e5b8e9b6c1d19c",
				Address:     "0x343c43a37d37dff08ae8c4a11544c718abb4fcf8",
				BlockNumber: blockNum + 10}

			Context("account exists", func() {
				It("returns the balance", func() {
					balanceString := "987654321098765432109876543210"
					balance, ok := new(big.Int).SetString(balanceString, 10)
					Expect(ok).Should(BeTrue())
					mockServiceManager.On("GetERC20Balance", ctx, gethCommon.HexToAddress(req.Token), gethCommon.HexToAddress(req.Address), req.BlockNumber).Return(balance, new(big.Int).SetInt64(blockNum), nil).Once()
					res, err := svr.GetBalance(ctx, req)
					Expect(err).Should(Succeed())
					expRes := &pb.GetBalanceResponse{Amount: balanceString, BlockNumber: blockNum}
					Expect(res).Should(Equal(expRes))
				})
			})

			Context("account does not exist", func() {
				It("returns error", func() {
					mockServiceManager.On("GetERC20Balance", ctx, gethCommon.HexToAddress(req.Token), gethCommon.HexToAddress(req.Address), req.BlockNumber).Return(nil, nil, gorm.ErrRecordNotFound).Once()
					res, err := svr.GetBalance(ctx, req)
					Expect(err).Should(Equal(ErrInternal))
					Expect(res).Should(BeNil())
				})
			})

			Context("transient error", func() {
				It("returns error", func() {
					mockServiceManager.On("GetERC20Balance", ctx, gethCommon.HexToAddress(req.Token), gethCommon.HexToAddress(req.Address), req.BlockNumber).Return(nil, nil, driver.ErrBadConn).Once()
					res, err := svr.GetBalance(ctx, req)
					Expect(err).Should(Equal(ErrInternal))
					Expect(res).Should(BeNil())
				})
			})
		})
	})

	Context("GetOffsetBalance()", func() {
		ctx := context.Background()
		header := &model.Header{
			Number: 5430200,
		}
		Context("bad Offset", func() {
			It("returns error due to large offset", func() {
				req := &pb.GetOffsetBalanceRequest{Offset: 54302000}
				mockServiceManager.On("FindLatestBlock").Return(header, nil).Once()
				res, err := svr.GetOffsetBalance(ctx, req)
				Expect(err).Should(Equal(ErrInvalidOffset))
				Expect(res).Should(BeNil())
			})
		})

		Context("ETH", func() {
			blockNum := int64(5430100)
			req := &pb.GetOffsetBalanceRequest{
				Token:   ethToken,
				Address: "0x343c43a37d37dff08ae8c4a11544c718abb4fcf8",
				Offset:  10,
			}
			target := header.Number - req.Offset

			Context("account exists", func() {
				It("returns the balance", func() {
					balanceString := "987654321098765432109876543210"
					balance, ok := new(big.Int).SetString(balanceString, 10)
					Expect(ok).Should(BeTrue())
					mockServiceManager.On("FindLatestBlock").Return(header, nil).Once()
					mockServiceManager.On("GetBalance", ctx, gethCommon.HexToAddress(req.Address), target).Return(balance, new(big.Int).SetInt64(blockNum), nil).Once()
					res, err := svr.GetOffsetBalance(ctx, req)
					Expect(err).Should(Succeed())
					expRes := &pb.GetBalanceResponse{Amount: balanceString, BlockNumber: blockNum}
					Expect(res).Should(Equal(expRes))
				})
			})

			Context("account does not exist", func() {
				It("returns error", func() {
					mockServiceManager.On("FindLatestBlock").Return(header, nil).Once()
					mockServiceManager.On("GetBalance", ctx, gethCommon.HexToAddress(req.Address), target).Return(nil, nil, gorm.ErrRecordNotFound).Once()
					res, err := svr.GetOffsetBalance(ctx, req)
					Expect(err).Should(Equal(ErrInternal))
					Expect(res).Should(BeNil())
				})
			})

			Context("transient error", func() {
				It("returns error", func() {
					mockServiceManager.On("FindLatestBlock").Return(header, nil).Once()
					mockServiceManager.On("GetBalance", ctx, gethCommon.HexToAddress(req.Address), target).Return(nil, nil, driver.ErrBadConn).Once()
					res, err := svr.GetOffsetBalance(ctx, req)
					Expect(err).Should(Equal(ErrInternal))
					Expect(res).Should(BeNil())
				})
			})
		})

		Context("ERC20", func() {
			blockNum := int64(5430100)
			req := &pb.GetOffsetBalanceRequest{
				Token:   "0xfffd933a0bc612844eaf0c6fe3e5b8e9b6c1d19c",
				Address: "0x343c43a37d37dff08ae8c4a11544c718abb4fcf8",
				Offset:  10,
			}

			target := header.Number - req.Offset
			Context("account exists", func() {
				It("returns the balance", func() {
					balanceString := "987654321098765432109876543210"
					balance, ok := new(big.Int).SetString(balanceString, 10)
					Expect(ok).Should(BeTrue())
					mockServiceManager.On("FindLatestBlock").Return(header, nil).Once()
					mockServiceManager.On("GetERC20Balance", ctx, gethCommon.HexToAddress(req.Token), gethCommon.HexToAddress(req.Address), target).Return(balance, new(big.Int).SetInt64(blockNum), nil).Once()
					res, err := svr.GetOffsetBalance(ctx, req)
					Expect(err).Should(Succeed())
					expRes := &pb.GetBalanceResponse{Amount: balanceString, BlockNumber: blockNum}
					Expect(res).Should(Equal(expRes))
				})
			})

			Context("account does not exist", func() {
				It("returns error", func() {
					mockServiceManager.On("FindLatestBlock").Return(header, nil).Once()
					mockServiceManager.On("GetERC20Balance", ctx, gethCommon.HexToAddress(req.Token), gethCommon.HexToAddress(req.Address), target).Return(nil, nil, gorm.ErrRecordNotFound).Once()
					res, err := svr.GetOffsetBalance(ctx, req)
					Expect(err).Should(Equal(ErrInternal))
					Expect(res).Should(BeNil())
				})
			})

			Context("transient error", func() {
				It("returns error", func() {
					mockServiceManager.On("FindLatestBlock").Return(header, nil).Once()
					mockServiceManager.On("GetERC20Balance", ctx, gethCommon.HexToAddress(req.Token), gethCommon.HexToAddress(req.Address), target).Return(nil, nil, driver.ErrBadConn).Once()
					res, err := svr.GetOffsetBalance(ctx, req)
					Expect(err).Should(Equal(ErrInternal))
					Expect(res).Should(BeNil())
				})
			})
		})
	})
})

func TestRpcServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Server RPC Test")
}
