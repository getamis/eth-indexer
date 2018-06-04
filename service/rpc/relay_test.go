// Copyright © 2018 AMIS Technologies
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package rpc

import (
	"context"
	"errors"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/getamis/sirius/log"
	"github.com/getamis/eth-indexer/client/mocks"
	"github.com/getamis/eth-indexer/contracts"
	. "github.com/getamis/eth-indexer/service"
	"github.com/getamis/eth-indexer/service/pb"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

var _ = Describe("Relay Server Test", func() {
	var (
		mockClient *mocks.EthClient
		svr        *relayServer
	)

	BeforeEach(func() {
		mockClient = new(mocks.EthClient)
		parsed, _ := abi.JSON(strings.NewReader(contracts.ERC20TokenABI))
		svr = &relayServer{
			logger:   log.New(),
			client:   mockClient,
			erc20ABI: parsed,
		}
	})

	AfterEach(func() {
		mockClient.AssertExpectations(GinkgoT())
	})

	Context("GetBlockByHash()", func() {
		ctx := context.Background()
		blockHashHex := "0x58bb59babd8fd8299b22acb997832a75d7b6b666579f80cc281764342f2b373b"
		req := &pb.BlockHashQueryRequest{Hash: blockHashHex}

		It("valid parameters", func() {
			// TODO: need more tests for a real block
			expBlock := types.NewBlockWithHeader(&types.Header{})
			mockClient.On("BlockByHash", ctx, ethCommon.HexToHash(blockHashHex)).Return(expBlock, nil).Once()
			res, err := svr.GetBlockByHash(ctx, req)
			Expect(err).Should(Succeed())
			Expect(res).ShouldNot(BeNil())
		})

		Context("invalid parameters", func() {
			unknownErr := errors.New("unknown error")
			It("failed to get block by hash", func() {
				mockClient.On("BlockByHash", ctx, ethCommon.HexToHash(blockHashHex)).Return(nil, unknownErr).Once()
				res, err := svr.GetBlockByHash(ctx, req)
				Expect(err).Should(Equal(ErrInternal))
				Expect(res).Should(BeNil())
			})
		})
	})

	Context("GetBlockByNumber()", func() {
		ctx := context.Background()
		blockNum := int64(1000300)
		req := &pb.BlockNumberQueryRequest{Number: blockNum}

		It("valid parameters", func() {
			// TODO: need more tests for a real block
			expBlock := types.NewBlockWithHeader(&types.Header{})
			mockClient.On("BlockByNumber", ctx, new(big.Int).SetInt64(req.Number)).Return(expBlock, nil).Once()
			res, err := svr.GetBlockByNumber(ctx, req)
			Expect(err).Should(Succeed())
			Expect(res).ShouldNot(BeNil())
		})

		Context("invalid parameters", func() {
			unknownErr := errors.New("unknown error")
			It("failed to get block by hash", func() {
				mockClient.On("BlockByNumber", ctx, new(big.Int).SetInt64(req.Number)).Return(nil, unknownErr).Once()
				res, err := svr.GetBlockByNumber(ctx, req)
				Expect(err).Should(Equal(ErrInternal))
				Expect(res).Should(BeNil())
			})
		})
	})

	Context("GetTransactionByHash()", func() {
		ctx := context.Background()
		txHashHex := "0x58bb59babd8fd8299b22acb997832a75d7b6b666579f80cc281764342f2b373b"
		req := &pb.TransactionQueryRequest{Hash: txHashHex}

		It("valid parameters", func() {
			// TODO: need more tests for a real tx
			expTx := types.NewTransaction(0, ethCommon.HexToAddress("0x01"), big.NewInt(10), 1, big.NewInt(11), []byte{})
			mockClient.On("TransactionByHash", ctx, ethCommon.HexToHash(txHashHex)).Return(expTx, false, nil).Once()
			res, err := svr.GetTransactionByHash(ctx, req)
			Expect(err).Should(Succeed())
			Expect(res).ShouldNot(BeNil())
		})

		Context("invalid parameters", func() {
			unknownErr := errors.New("unknown error")
			It("failed to get block by hash", func() {
				mockClient.On("TransactionByHash", ctx, ethCommon.HexToHash(txHashHex)).Return(nil, false, unknownErr).Once()
				res, err := svr.GetTransactionByHash(ctx, req)
				Expect(err).Should(Equal(ErrInternal))
				Expect(res).Should(BeNil())
			})
		})
	})

	Context("GetBalance()", func() {
		ctx := context.Background()
		req := &pb.GetBalanceRequest{
			Address:     "0x343c43a37d37dff08ae8c4a11544c718abb4fcf8",
			BlockNumber: int64(1000300),
		}
		Context("eth token", func() {
			It("valid parameters", func() {
				req.Token = ethToken
				balance := big.NewInt(100)
				expRes := &pb.GetBalanceResponse{
					Amount:      balance.String(),
					BlockNumber: req.BlockNumber,
				}
				mockClient.On("BalanceAt", ctx, ethCommon.HexToAddress(req.Address), new(big.Int).SetInt64(req.BlockNumber)).Return(balance, nil).Once()
				res, err := svr.GetBalance(ctx, req)
				Expect(err).Should(Succeed())
				Expect(res).Should(Equal(expRes))
			})
			It("valid parameters (block number = -1)", func() {
				var number *big.Int
				balance := big.NewInt(100)
				expRes := &pb.GetBalanceResponse{
					Amount:      balance.String(),
					BlockNumber: int64(1000301),
				}
				mockClient.On("BlockByNumber", ctx, number).Return(types.NewBlockWithHeader(
					&types.Header{
						Number: big.NewInt(expRes.BlockNumber),
					},
				), nil).Once()
				mockClient.On("BalanceAt", ctx, ethCommon.HexToAddress(req.Address), new(big.Int).SetInt64(expRes.BlockNumber)).Return(balance, nil).Once()
				res, err := svr.GetBalance(ctx, &pb.GetBalanceRequest{
					Address:     "0x343c43a37d37dff08ae8c4a11544c718abb4fcf8",
					BlockNumber: -1,
					Token:       ethToken,
				})
				Expect(err).Should(Succeed())
				Expect(res).Should(Equal(expRes))
			})
			Context("invalid parameters", func() {
				unknownErr := errors.New("unknown error")
				It("failed to call balance at", func() {
					req.Token = ethToken
					mockClient.On("BalanceAt", ctx, ethCommon.HexToAddress(req.Address), new(big.Int).SetInt64(req.BlockNumber)).Return(nil, unknownErr).Once()
					res, err := svr.GetBalance(ctx, req)
					Expect(err).Should(Equal(ErrInternal))
					Expect(res).Should(BeNil())
				})
			})
		})

		Context("other tokens", func() {
			It("valid parameters", func() {
				expectedBalance := "1"

				// output for getBalance
				output := []uint8{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 13, 224, 182, 179, 167, 100, 0, 0}

				// output for decimal
				decimalOutput := []uint8{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 18}

				req.Token = "0x3893b9422cd5d70a81edeffe3d5a1c6a978310bb"
				req.Address = "0xccbf4a59bc42129dcb80a8b703dcf4217635a91d"
				req.BlockNumber = 5141437
				mockClient.On("CallContract", ctx, mock.Anything, new(big.Int).SetInt64(req.BlockNumber)).Return(output, nil).Once()
				mockClient.On("CallContract", ctx, mock.Anything, new(big.Int).SetInt64(req.BlockNumber)).Return(decimalOutput, nil).Once()
				res, err := svr.GetBalance(ctx, req)
				Expect(err).Should(BeNil())
				Expect(res.Amount).Should(Equal(expectedBalance))
			})
			Context("invalid parameters", func() {
				unknownErr := errors.New("unknown error")
				It("failed to call contract", func() {
					req.Token = "0x343c43a37d37dff08ae8c4a11544c718abb4fcf9"
					mockClient.On("CallContract", ctx, mock.Anything, new(big.Int).SetInt64(req.BlockNumber)).Return(nil, unknownErr).Once()
					res, err := svr.GetBalance(ctx, req)
					Expect(err).Should(Equal(ErrInternal))
					Expect(res).Should(BeNil())
				})
			})
		})
	})

	Context("GetOffsetBalance()", func() {
		var nilInt *big.Int
		ctx := context.Background()
		req := &pb.GetOffsetBalanceRequest{
			Address: "0x343c43a37d37dff08ae8c4a11544c718abb4fcf8",
			Offset:  int64(10),
		}
		Context("eth token", func() {
			It("valid parameters", func() {
				req.Token = ethToken
				balance := big.NewInt(100)
				block := types.NewBlockWithHeader(&types.Header{
					Number: big.NewInt(101),
				})
				target := big.NewInt(block.Number().Int64() - req.Offset)
				expRes := &pb.GetBalanceResponse{
					Amount:      balance.String(),
					BlockNumber: target.Int64(),
				}
				mockClient.On("BlockByNumber", ctx, nilInt).Return(block, nil).Once()
				mockClient.On("BalanceAt", ctx, ethCommon.HexToAddress(req.Address), target).Return(balance, nil).Once()
				res, err := svr.GetOffsetBalance(ctx, req)
				Expect(err).Should(Succeed())
				Expect(res).Should(Equal(expRes))
			})
			Context("invalid parameters", func() {
				unknownErr := errors.New("unknown error")
				It("failed to call balance at", func() {
					req.Token = ethToken
					block := types.NewBlockWithHeader(&types.Header{
						Number: big.NewInt(101),
					})
					target := big.NewInt(block.Number().Int64() - req.Offset)
					mockClient.On("BlockByNumber", ctx, nilInt).Return(block, nil).Once()
					mockClient.On("BalanceAt", ctx, ethCommon.HexToAddress(req.Address), target).Return(nil, unknownErr).Once()
					res, err := svr.GetOffsetBalance(ctx, req)
					Expect(err).Should(Equal(ErrInternal))
					Expect(res).Should(BeNil())
				})

				It("failed due to large offset", func() {
					req.Token = ethToken
					req.Offset = 102
					block := types.NewBlockWithHeader(&types.Header{
						Number: big.NewInt(101),
					})
					mockClient.On("BlockByNumber", ctx, nilInt).Return(block, nil).Once()
					res, err := svr.GetOffsetBalance(ctx, req)
					Expect(err).Should(Equal(ErrInvalidOffset))
					Expect(res).Should(BeNil())
				})

				It("failed to get block", func() {
					req.Token = ethToken
					mockClient.On("BlockByNumber", ctx, nilInt).Return(nil, unknownErr).Once()
					res, err := svr.GetOffsetBalance(ctx, req)
					Expect(err).Should(Equal(ErrInternal))
					Expect(res).Should(BeNil())
				})
			})
		})

		Context("other tokens", func() {
			// TODO: mock CallContract correctly
			// It("valid parameters", func() {
			// 	balance := big.NewInt(100)
			// 	expRes := &pb.GetBalanceResponse{
			// 		Amount:      balance.String(),
			// 		BlockNumber: req.BlockNumber,
			// 	}
			// 	mockClient.On("CallContract", ctx, mock.Anything, new(big.Int).SetInt64(req.BlockNumber)).Return(balance.Bytes(), nil).Once()
			// 	res, err := svr.GetBalance(ctx, req)
			// 	Expect(err).Should(Succeed())
			// 	Expect(res).Should(Equal(expRes))
			// })
			Context("invalid parameters", func() {
				unknownErr := errors.New("unknown error")
				It("failed to call contract", func() {
					req.Offset = 10
					req.Token = "0x343c43a37d37dff08ae8c4a11544c718abb4fcf9"
					block := types.NewBlockWithHeader(&types.Header{
						Number: big.NewInt(101),
					})
					target := big.NewInt(block.Number().Int64() - req.Offset)
					mockClient.On("BlockByNumber", ctx, nilInt).Return(block, nil).Once()
					mockClient.On("CallContract", ctx, mock.Anything, target).Return(nil, unknownErr).Once()
					res, err := svr.GetOffsetBalance(ctx, req)
					Expect(err).Should(Equal(ErrInternal))
					Expect(res).Should(BeNil())
				})
				It("failed due to large offset", func() {
					req.Token = "0x343c43a37d37dff08ae8c4a11544c718abb4fcf9"
					req.Offset = 102
					block := types.NewBlockWithHeader(&types.Header{
						Number: big.NewInt(101),
					})
					mockClient.On("BlockByNumber", ctx, nilInt).Return(block, nil).Once()
					res, err := svr.GetOffsetBalance(ctx, req)
					Expect(err).Should(Equal(ErrInvalidOffset))
					Expect(res).Should(BeNil())
				})
				It("failed to call contract", func() {
					req.Offset = 10
					req.Token = "0x343c43a37d37dff08ae8c4a11544c718abb4fcf9"
					mockClient.On("BlockByNumber", ctx, nilInt).Return(nil, unknownErr).Once()
					res, err := svr.GetOffsetBalance(ctx, req)
					Expect(err).Should(Equal(ErrInternal))
					Expect(res).Should(BeNil())
				})
			})
		})
	})
})
