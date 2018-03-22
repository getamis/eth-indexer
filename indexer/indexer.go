package indexer

import (
	"github.com/getamis/sirius/log"
	store "github.com/maichain/eth-indexer/store/block_header"
)

type Indexer interface {
	Start()
}

func NewIndexer(store store.Store) Indexer {
	return &indexer{
		store,
	}
}

type indexer struct {
	store store.Store
}

func (i *indexer) Start() {
	log.Info("Start Indexing")
}
