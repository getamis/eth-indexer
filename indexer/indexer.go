package indexer

import (
	"context"
	"math/big"

	"github.com/getamis/sirius/log"
	store "github.com/maichain/eth-indexer/store/block_header"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/getamis/sirius/log"
)

var logger = log.New()

type Indexer interface {
	Start()
}

func NewIndexer(client *ethclient.Client, store store.Store) Indexer {
	return &indexer{
		client,
		store,
	}
}

type indexer struct {
	client *ethclient.Client
	store  store.Store
}

func (indexer *indexer) Start() error {
	ctx := context.TODO()

	start := big.NewInt(1)
	end := big.NewInt(10)
	for i := new(big.Int).Set(start); i.Cmp(end) <= 0; i.Add(i, big.NewInt(1)) {
		block, err := indexer.client.BlockByNumber(ctx, i)
		if err != nil {
			return err
		}
		logger.Info(block.Hash().String())
	}
	return nil
}
