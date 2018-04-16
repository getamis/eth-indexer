package indexer

import (
	"context"
	"math/big"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/getamis/sirius/log"
	"github.com/maichain/eth-indexer/service/pb"
	"github.com/maichain/eth-indexer/store"
)

//go:generate mockery -name EthClient

type EthClient interface {
	BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error)
	TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error)
	SubscribeNewHead(ctx context.Context, ch chan<- *types.Header) (ethereum.Subscription, error)
}

// New news an indexer service
func New(client EthClient, storeManager store.Manager) *indexer {
	return &indexer{
		client:  client,
		manager: storeManager,
	}
}

type indexer struct {
	client  EthClient
	manager store.Manager
}

func (idx *indexer) Listen(ctx context.Context, ch chan *types.Header) error {
	childCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Get latest header from db
	header, err := idx.manager.LatestHeader()
	if err != nil {
		if store.NotFoundError(err) {
			log.Info("The header db is empty")
			header = &pb.BlockHeader{
				Number: -1,
			}
		} else {
			log.Error("Failed to get latest header from db", "err", err)
			return err
		}
	}

	// Get latest blocks from ethereum
	latestBlock, err := idx.client.BlockByNumber(ctx, nil)
	if err != nil {
		log.Error("Failed to get latest header from ethereum", "err", err)
		return err
	}

	// Sync missing blocks from ethereum
	latestBlock, err = idx.sync(childCtx, header.Number, header.Hash, latestBlock.Number().Int64())
	if err != nil {
		log.Error("Failed to sync to latest blocks from ethereum", "from", header.Number, "fromHash", header.Hash, "err", err)
		return err
	}

	// Listen new channel events
	_, err = idx.client.SubscribeNewHead(childCtx, ch)
	if err != nil {
		log.Error("Failed to subscribe event for new header from ethereum", "err", err)
		return err
	}

	for {
		select {
		case head := <-ch:
			log.Trace("Got new header", "number", head.Number, "hash", store.HashHex(head.Hash()))
			latestBlock, err = idx.sync(childCtx, latestBlock.Number().Int64(), store.HashHex(latestBlock.Hash()), head.Number.Int64())
			if err != nil {
				log.Error("Failed to sync to blocks from ethereum", "from", latestBlock.Number, "fromHash", latestBlock.Hash, "to", head.Number.Int64(), "err", err)
				return err
			}
		case <-childCtx.Done():
			return childCtx.Err()
		}
	}
}

// sync syncs the blocks and header into database
func (idx *indexer) sync(ctx context.Context, from int64, fromHash string, to int64) (block *types.Block, err error) {
	// Update existing blocks from ethereum to db
	for i := from + 1; i <= to; i++ {
		block, err = idx.client.BlockByNumber(ctx, big.NewInt(i))
		if err != nil {
			log.Error("Failed to get block from ethereum", "number", i, "err", err)
			return nil, err
		}

		// TODO: How to handle fork case
		// Check whether fork happens
		// if prevHash != utils.Hex(block.Hash()) {
		//
		// } else {
		// }

		var receipts []*types.Receipt
		for _, tx := range block.Transactions() {
			r, err := idx.client.TransactionReceipt(ctx, tx.Hash())
			if err != nil {
				log.Error("Failed to get receipt from ethereum", "number", i, "tx", tx.Hash(), "err", err)
				return nil, err
			}
			receipts = append(receipts, r)
		}

		err = idx.manager.InsertBlock(block, receipts)
		if err != nil {
			log.Error("Failed to insert block", "number", i, "err", err)
			return
		}
		log.Trace("Inserted block", "number", i, "hash", store.HashHex(block.Hash()), "txs", len(block.Transactions()))
	}
	return
}
