package indexer

import (
	"context"
	"testing"
	"time"

	common "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	rlp "github.com/ethereum/go-ethereum/rlp"
	rpc "github.com/ethereum/go-ethereum/rpc"
	indexerMocks "github.com/maichain/eth-indexer/service/indexer/mocks"
	ManagerMocks "github.com/maichain/eth-indexer/store/store_manager/mocks"
	"github.com/stretchr/testify/mock"
)

func TestListen(t *testing.T) {
	mockEthClient := new(indexerMocks.EthClient)
	mockManager := new(ManagerMocks.StoreManager)

	const expectedBlockNumber int = 1

	/* this block copy from block_test.go in go-ethereum repository
	 * with block Number 1
	 */
	blockEnc := common.FromHex("f90260f901f9a083cafc574e1f51ba9dc0568fc617a08ea2429fb384059c972f13b19fa1c8dd55a01dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347948888f1f195afa192cfee860698584c030f4c9db1a0ef1552a40b7165c3cd773806b9e0c165b75356e0314bf0706f279c729f51e017a05fe50b260da6308036625b850b5d6ced6d0a9f814c0688bc91ffb7b7a3a54b67a0bc37d79753ad738a6dac4921e57392f145d8887476de3f783dfa7edae9283e52b90100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000008302000001832fefd8825208845506eb0780a0bd4472abb6659ebe3ee06ee4d7b72a00a9f4d001caca51342001075469aff49888a13a5a8c8f2bb1c4f861f85f800a82c35094095e7baea6a6c7c4c2dfeb977efac326af552d870a801ba09bea4c4daac7c7c52e093e6a4c35dbbcf8856f1af7b059ba20253e70848d094fa08a8fae537ce25ed8cb5af9adac3f141af69bd515bd2ba031522df09b97dd72b1c0")
	var block types.Block
	rlp.DecodeBytes(blockEnc, &block)

	receipt := new(types.Receipt)
	sub := &rpc.ClientSubscription{}

	mockEthClient.On("BlockByNumber", mock.Anything, mock.Anything).Return(&block, nil)
	mockEthClient.On("TransactionReceipt", mock.Anything, mock.Anything).Return(receipt, nil)
	mockEthClient.On("SubscribeNewHead", mock.Anything, mock.Anything).Return(sub, nil)

	mockManager.On("GetLatestHeader").Return(nil, nil).Once()

	// Called twice since we have block 0 and 1
	mockManager.On("Upsert", mock.Anything, mock.Anything, mock.Anything).Return(nil).Twice()

	ch := make(chan *types.Header)
	ctx, cancel := context.WithCancel(context.Background())

	indexer := NewIndexer(mockEthClient, mockManager)
	go func() {
		indexer.Listen(ctx, ch)
	}()

	ch <- block.Header()

	// Wait for race condition since need to wait passing header to Listen function via channel
	time.Sleep(time.Second)
	cancel()
	mockManager.AssertExpectations(t)
}
