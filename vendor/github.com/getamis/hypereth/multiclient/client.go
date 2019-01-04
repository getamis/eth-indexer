// Copyright 2018 AMIS Technologies
// This file is part of the hypereth library.
//
// The hypereth library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The hypereth library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the hypereth library. If not, see <http://www.gnu.org/licenses/>.

package multiclient

import (
	"context"
	"errors"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/getamis/sirius/log"
)

const (
	dialTimeout = 5 * time.Second
)

var (
	ErrInvalidTypeCast = errors.New("invalid type cast")
	ErrNoEthClient     = errors.New("no eth client")

	retryPeriod = 10 * time.Second
)

type Client struct {
	rpcClientMap       *Map
	subscribeNewHeadWg sync.WaitGroup
}

func New(ctx context.Context, opts ...Option) (*Client, error) {
	mc := &Client{
		rpcClientMap: NewMap(),
	}
	for _, opt := range opts {
		if err := opt(mc); err != nil {
			return nil, err
		}
	}

	urls := mc.rpcClientMap.Keys()
	lens := len(urls)
	if lens == 0 {
		return nil, ErrNoEthClient
	}

	log.Debug("Create multiclient", "urls", lens)
	errCh := make(chan error, lens)
	for _, rawURL := range urls {
		go func(ctx context.Context, rawURL string) {
			ctx, cancel := context.WithTimeout(ctx, dialTimeout)
			defer cancel()
			c, err := rpc.DialContext(ctx, rawURL)
			if err == nil {
				log.Info("Connect to ethclient successfully", "url", rawURL)
				mc.rpcClientMap.Set(rawURL, c)
			} else {
				log.Error("Failed to dial eth client", "rawURL", rawURL, "err", err)
			}
			errCh <- err
		}(ctx, rawURL)
	}

	var dialErr error
	for i := 0; i < lens; i++ {
		err := <-errCh
		if err != nil {
			dialErr = err
		}
	}
	if dialErr != nil {
		mc.Close()
		return nil, dialErr
	}
	return mc, nil
}

// Close closes an existing RPC connection.
func (mc *Client) Close() {
	clients := mc.rpcClientMap.List()
	for _, c := range clients {
		c.Close()
	}
}

func (mc *Client) EthClients() []*ethclient.Client {
	clients := mc.rpcClientMap.List()
	ethClients := make([]*ethclient.Client, len(clients))
	for i, c := range clients {
		ethClients[i] = ethclient.NewClient(c)
	}
	return ethClients
}

func (mc *Client) RPCClients() []*rpc.Client {
	return mc.rpcClientMap.List()
}

// Blockchain Access

// BlockByHash returns the given full block.
//
// Note that loading full blocks requires two requests. Use HeaderByHash
// if you don't need all transactions or uncle headers.
func (mc *Client) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	clients := mc.rpcClientMap.List()
	if len(clients) == 0 {
		return nil, ErrNoEthClient
	}
	fns := make([]getFn, len(clients))
	for i, c := range clients {
		ec := ethclient.NewClient(c)
		fns[i] = func(ctx context.Context) (interface{}, error) {
			return ec.BlockByHash(ctx, hash)
		}
	}

	resp, err := getFromAny(ctx, fns)
	if err != nil {
		log.Warn("Failed to get block by hash from any eth client", "hash", hash.Hex(), "err", err)
		return nil, err
	}
	block, ok := resp.(*types.Block)
	if !ok {
		log.Error("Failed to cast type to *types.Block")
		return nil, ErrInvalidTypeCast
	}
	return block, nil
}

// BlockByNumber returns a block from the current canonical chain. If number is nil, the
// latest known block is returned.
//
// Note that loading full blocks requires two requests. Use HeaderByNumber
// if you don't need all transactions or uncle headers.
func (mc *Client) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	clients := mc.rpcClientMap.List()
	if len(clients) == 0 {
		return nil, ErrNoEthClient
	}
	fns := make([]getFn, len(clients))
	for i, c := range clients {
		ec := ethclient.NewClient(c)
		fns[i] = func(ctx context.Context) (interface{}, error) {
			return ec.BlockByNumber(ctx, number)
		}
	}

	resp, err := getFromAny(ctx, fns)
	if err != nil {
		log.Warn("Failed to get block by number from any eth client", "number", number.String(), "err", err)
		return nil, err
	}
	block, ok := resp.(*types.Block)
	if !ok {
		log.Error("Failed to cast type to *types.Block")
		return nil, ErrInvalidTypeCast
	}
	return block, nil
}

// HeaderByHash returns the block header with the given hash.
func (mc *Client) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	clients := mc.rpcClientMap.List()
	if len(clients) == 0 {
		return nil, ErrNoEthClient
	}
	fns := make([]getFn, len(clients))
	for i, c := range clients {
		ec := ethclient.NewClient(c)
		fns[i] = func(ctx context.Context) (interface{}, error) {
			return ec.HeaderByHash(ctx, hash)
		}
	}

	resp, err := getFromAny(ctx, fns)
	if err != nil {
		log.Warn("Failed to get block header by hash from any eth client", "hash", hash.Hex(), "err", err)
		return nil, err
	}
	head, ok := resp.(*types.Header)
	if !ok {
		log.Error("Failed to cast type to *types.Header")
		return nil, ErrInvalidTypeCast
	}
	return head, nil
}

// HeaderByNumber returns a block header from the current canonical chain. If number is
// nil, the latest known header is returned.
func (mc *Client) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	clients := mc.rpcClientMap.List()
	if len(clients) == 0 {
		return nil, ErrNoEthClient
	}
	fns := make([]getFn, len(clients))
	for i, c := range clients {
		ec := ethclient.NewClient(c)
		fns[i] = func(ctx context.Context) (interface{}, error) {
			return ec.HeaderByNumber(ctx, number)
		}
	}

	resp, err := getFromAny(ctx, fns)
	if err != nil {
		log.Warn("Failed to get block header by number from any eth client", "number", number.String(), "err", err)
		return nil, err
	}
	head, ok := resp.(*types.Header)
	if !ok {
		log.Error("Failed to cast type to *types.Header")
		return nil, ErrInvalidTypeCast
	}
	return head, nil
}

type txResponse struct {
	tx        *types.Transaction
	isPending bool
	err       error
}

// TransactionByHash returns the transaction with the given hash.
func (mc *Client) TransactionByHash(ctx context.Context, hash common.Hash) (tx *types.Transaction, isPending bool, err error) {
	clients := mc.rpcClientMap.List()
	if len(clients) == 0 {
		return nil, false, ErrNoEthClient
	}
	getFromAny := func(ctx context.Context, hash common.Hash) (*types.Transaction, bool, error) {
		respCh := make(chan *txResponse, len(clients))
		for _, c := range clients {
			go func(ec *ethclient.Client) {
				tx, isPending, err := ec.TransactionByHash(ctx, hash)
				if err != nil {
					respCh <- &txResponse{err: err}
					return
				}
				respCh <- &txResponse{tx: tx, isPending: isPending}
			}(ethclient.NewClient(c))
		}
		var resp *txResponse
		for i := 0; i < len(clients); i++ {
			resp = <-respCh
			if resp.err == nil {
				break
			}
		}
		return resp.tx, resp.isPending, resp.err
	}

	tx, isPending, err = getFromAny(ctx, hash)
	if err != nil {
		log.Warn("Failed to get transaction by hash from any eth client", "hash", hash.Hex(), "err", err)
	}
	return
}

// State Access

// BalanceAt returns the wei balance of the given account.
// The block number can be nil, in which case the balance is taken from the latest known block.
func (mc *Client) BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error) {
	clients := mc.rpcClientMap.List()
	if len(clients) == 0 {
		return nil, ErrNoEthClient
	}
	fns := make([]getFn, len(clients))
	for i, c := range clients {
		ec := ethclient.NewClient(c)
		fns[i] = func(ctx context.Context) (interface{}, error) {
			return ec.BalanceAt(ctx, account, blockNumber)
		}
	}

	resp, err := getFromAny(ctx, fns)
	if err != nil {
		log.Warn("Failed to get balance from any eth client", "account", account.Hex(), "blockNumber", blockNumber.String(), "err", err)
		return nil, err
	}
	balance, ok := resp.(*big.Int)
	if !ok {
		log.Error("Failed to cast type to *big.Int")
		return nil, ErrInvalidTypeCast
	}
	return balance, nil
}

// CodeAt returns the contract code of the given account.
// The block number can be nil, in which case the code is taken from the latest known block.
func (mc *Client) CodeAt(ctx context.Context, account common.Address, blockNumber *big.Int) ([]byte, error) {
	clients := mc.rpcClientMap.List()
	if len(clients) == 0 {
		return nil, ErrNoEthClient
	}
	fns := make([]getFn, len(clients))
	for i, c := range clients {
		ec := ethclient.NewClient(c)
		fns[i] = func(ctx context.Context) (interface{}, error) {
			return ec.CodeAt(ctx, account, blockNumber)
		}
	}

	resp, err := getFromAny(ctx, fns)
	if err != nil {
		log.Warn("Failed to get code from any eth client", "account", account.Hex(), "blockNumber", blockNumber.String(), "err", err)
		return nil, err
	}
	code, ok := resp.([]byte)
	if !ok {
		log.Error("Failed to cast type to []byte")
		return nil, ErrInvalidTypeCast
	}
	return code, nil
}

// NonceAt returns the account nonce of the given account.
// The block number can be nil, in which case the nonce is taken from the latest known block.
func (mc *Client) NonceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (uint64, error) {
	clients := mc.rpcClientMap.List()
	if len(clients) == 0 {
		return 0, ErrNoEthClient
	}
	fns := make([]getFn, len(clients))
	for i, c := range clients {
		ec := ethclient.NewClient(c)
		fns[i] = func(ctx context.Context) (interface{}, error) {
			return ec.NonceAt(ctx, account, blockNumber)
		}
	}

	resp, err := getFromAny(ctx, fns)
	if err != nil {
		log.Warn("Failed to get nonce from any eth client", "account", account.Hex(), "blockNumber", blockNumber.String(), "err", err)
		return uint64(0), err
	}
	nonce, ok := resp.(uint64)
	if !ok {
		log.Error("Failed to cast type to uint64")
		return uint64(0), ErrInvalidTypeCast
	}
	return nonce, nil
}

// Pending State

// PendingBalanceAt returns the wei balance of the given account in the pending state.
func (mc *Client) PendingBalanceAt(ctx context.Context, account common.Address) (*big.Int, error) {
	clients := mc.rpcClientMap.List()
	if len(clients) == 0 {
		return nil, ErrNoEthClient
	}
	fns := make([]getFn, len(clients))
	for i, c := range clients {
		ec := ethclient.NewClient(c)
		fns[i] = func(ctx context.Context) (interface{}, error) {
			return ec.PendingBalanceAt(ctx, account)
		}
	}

	resp, err := getFromAny(ctx, fns)
	if err != nil {
		log.Warn("Failed to get pending balance from any eth client", "account", account.Hex(), "err", err)
		return nil, err
	}
	balance, ok := resp.(*big.Int)
	if !ok {
		log.Error("Failed to cast type to *big.Int")
		return nil, ErrInvalidTypeCast
	}
	return balance, nil
}

// PendingNonceAt returns the account nonce of the given account in the pending state.
// This is the nonce that should be used for the next transaction.
func (mc *Client) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	clients := mc.rpcClientMap.List()
	if len(clients) == 0 {
		return 0, ErrNoEthClient
	}
	fns := make([]getFn, len(clients))
	for i, c := range clients {
		ec := ethclient.NewClient(c)
		fns[i] = func(ctx context.Context) (interface{}, error) {
			return ec.PendingNonceAt(ctx, account)
		}
	}

	resp, err := getFromAny(ctx, fns)
	if err != nil {
		log.Warn("Failed to get pending nonce from any eth client", "account", account.Hex(), "err", err)
		return uint64(0), err
	}
	nonce, ok := resp.(uint64)
	if !ok {
		log.Error("Failed to cast type to uint64")
		return uint64(0), ErrInvalidTypeCast
	}
	return nonce, nil
}

// Contract Calling

// CallContract executes a message call transaction, which is directly executed in the VM
// of the node, but never mined into the blockchain.
//
// blockNumber selects the block height at which the call runs. It can be nil, in which
// case the code is taken from the latest known block. Note that state from very old
// blocks might not be available.
func (mc *Client) CallContract(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	clients := mc.rpcClientMap.List()
	if len(clients) == 0 {
		return nil, ErrNoEthClient
	}
	fns := make([]getFn, len(clients))
	for i, c := range clients {
		ec := ethclient.NewClient(c)
		fns[i] = func(ctx context.Context) (interface{}, error) {
			return ec.CallContract(ctx, msg, blockNumber)
		}
	}

	resp, err := getFromAny(ctx, fns)
	if err != nil {
		log.Warn("Failed to call contract from any eth client", "from", msg.From.Hex(), "to", msg.To.Hex(), "blockNumber", blockNumber.String(), "err", err)
		return nil, err
	}
	hex, ok := resp.([]byte)
	if !ok {
		log.Error("Failed to cast type to []byte")
		return nil, ErrInvalidTypeCast
	}
	return hex, nil
}

// PendingCallContract executes a message call transaction using the EVM.
// The state seen by the contract call is the pending state.
func (mc *Client) PendingCallContract(ctx context.Context, msg ethereum.CallMsg) ([]byte, error) {
	clients := mc.rpcClientMap.List()
	if len(clients) == 0 {
		return nil, ErrNoEthClient
	}
	fns := make([]getFn, len(clients))
	for i, c := range clients {
		ec := ethclient.NewClient(c)
		fns[i] = func(ctx context.Context) (interface{}, error) {
			return ec.PendingCallContract(ctx, msg)
		}
	}

	resp, err := getFromAny(ctx, fns)
	if err != nil {
		log.Warn("Failed to call contract with pending state from any eth client", "from", msg.From.Hex(), "to", msg.To.Hex(), "err", err)
		return nil, err
	}
	hex, ok := resp.([]byte)
	if !ok {
		log.Error("Failed to cast type to []byte")
		return nil, ErrInvalidTypeCast
	}
	return hex, nil
}

// SendTransaction injects a signed transaction into the pending pool for execution.
//
// If the transaction was a contract creation use the TransactionReceipt method to get the
// contract address after the transaction has been mined.
func (mc *Client) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	clients := mc.rpcClientMap.List()
	if len(clients) == 0 {
		return ErrNoEthClient
	}
	fns := make([]postFn, len(clients))
	for i, c := range clients {
		ec := ethclient.NewClient(c)
		fns[i] = func(ctx context.Context) error {
			return ec.SendTransaction(ctx, tx)
		}
	}

	err := postToAll(ctx, fns)
	if err != nil {
		log.Warn("Failed to send transaction to any eth client", "txHash", tx.Hash().Hex(), "err", err)
		return err
	}
	return nil
}

// CallContext performs a JSON-RPC call with the given arguments. If the context is
// canceled before the call has successfully returned, CallContext returns immediately.
//
// The result must be a pointer so that package json can unmarshal into it. You
// can also pass nil, in which case the result is ignored.
//
// `isPostToAll` is true means waiting until received all responsese of JSON-RPC calls and return error if failed to perform JSON-RPC call to all eth client.
// Otherwise, just waiting for the first successful response and return error if failed to perform JSON-RPC call to all eth client.
// FIXME: The `result` may be overrode to wrong value because we share a `result` in all goroutines.
func (mc *Client) CallContext(ctx context.Context, isPostToAll bool, result interface{}, method string, args ...interface{}) error {
	clients := mc.rpcClientMap.List()
	if len(clients) == 0 {
		return ErrNoEthClient
	}
	var err error
	if !isPostToAll {
		fns := make([]getFn, len(clients))
		for i := range clients {
			c := clients[i]
			fns[i] = func(ctx context.Context) (interface{}, error) {
				err := c.CallContext(ctx, result, method, args...)
				return nil, err
			}
		}
		_, err = getFromAny(ctx, fns)
	} else {
		fns := make([]postFn, len(clients))
		for i := range clients {
			c := clients[i]
			fns[i] = func(ctx context.Context) error {
				return c.CallContext(ctx, result, method, args...)
			}
		}
		err = postToAll(ctx, fns)
	}

	if err != nil {
		log.Warn("Failed to perform a JSON-RPC call on any eth client", "err", err)
		return err
	}

	return nil
}

// BatchCall sends all given requests as a single batch and waits for the server
// to return a response for all of them. The wait duration is bounded by the
// context's deadline.
//
// In contrast to CallContext, BatchCallContext only returns errors that have occurred
// while sending the request. Any error specific to a request is reported through the
// Error field of the corresponding BatchElem.
//
// Note that batch calls may not be executed atomically on the server side.
//
// `isPostToAll` is true means waiting until received all responsese of JSON-RPC calls and return error if failed to perform JSON-RPC call to all eth client.
// Otherwise, just waiting for the first successful response and return error if failed to perform JSON-RPC call to all eth client.
// FIXME: The result may be overrode to wrong value because we share a `result` in all goroutines.
func (mc *Client) BatchCallContext(ctx context.Context, isPostToAll bool, b []rpc.BatchElem) error {
	var err error
	clients := mc.rpcClientMap.List()
	if len(clients) == 0 {
		return ErrNoEthClient
	}
	if !isPostToAll {
		fns := make([]getFn, len(clients))
		for i := range clients {
			c := clients[i]
			fns[i] = func(ctx context.Context) (interface{}, error) {
				err := c.BatchCallContext(ctx, b)
				return nil, err
			}
		}
		_, err = getFromAny(ctx, fns)
	} else {
		fns := make([]postFn, len(clients))
		for i := range clients {
			c := clients[i]
			fns[i] = func(ctx context.Context) error {
				return c.BatchCallContext(ctx, b)
			}
		}
		err = postToAll(ctx, fns)
	}

	if err != nil {
		log.Warn("Failed to perform batch JSON-RPC calls on any eth client", "err", err)
		return err
	}
	return nil
}

// Subscribe API
type Header struct {
	*types.Header
	*rpc.Client
}

// SubscribeNewHead subscribes to notifications about the current blockchain head
// on the given channel.
func (mc *Client) SubscribeNewHead(ctx context.Context, ch chan<- *Header) (ethereum.Subscription, error) {
	clientsMap := mc.rpcClientMap.Map()
	lens := len(clientsMap)
	if lens == 0 {
		return nil, ErrNoEthClient
	}

	cctx, cancel := context.WithCancel(ctx)
	for url := range clientsMap {
		go mc.subscribeNewHead(cctx, url, ch)
	}

	// TODO: handle new clients comes
	return event.NewSubscription(func(unsub <-chan struct{}) error {
		<-unsub
		cancel()
		mc.subscribeNewHeadWg.Wait()
		return nil
	}), nil
}

func (mc *Client) subscribeNewHead(ctx context.Context, url string, ch chan<- *Header) error {
	mc.subscribeNewHeadWg.Add(1)
	defer mc.subscribeNewHeadWg.Done()

	for {
		rc := mc.rpcClientMap.Get(url)
		if rc == nil {
			log.Trace("EthClient has been removed", "url", url)
			return nil
		}

		headerCh := make(chan *types.Header)
		c := ethclient.NewClient(rc)
		sub, err := c.SubscribeNewHead(ctx, headerCh)
		if err != nil {
			log.Warn("Failed to subscribe new head", "url", url, "err", err)
		} else {
		WAIT_NEW_HEADER:
			for {
				select {
				case header := <-headerCh:
					ch <- &Header{
						Client: rc,
						Header: header,
					}
				case err := <-sub.Err():
					log.Warn("Failed during subscription", "url", url, "err", err)
					break WAIT_NEW_HEADER
				case <-ctx.Done():
					return nil
				}
			}
		}
		// retry subscribe after retryPeriod
		time.Sleep(retryPeriod)
		log.Trace("Retry to subscribe new head", "url", url)
	}
}

type getFn func(ctx context.Context) (interface{}, error)

type getResponse struct {
	data interface{}
	err  error
}

func getFromAny(ctx context.Context, fns []getFn) (interface{}, error) {
	respCh := make(chan *getResponse, len(fns))
	getCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	for _, fn := range fns {
		go func(fn getFn) {
			data, err := fn(getCtx)
			if err != nil {
				respCh <- &getResponse{err: err}
				return
			}
			respCh <- &getResponse{data: data}
		}(fn)
	}

	var resp *getResponse
	for i := 0; i < len(fns); i++ {
		resp = <-respCh
		if resp.err == nil {
			break
		}
	}
	return resp.data, resp.err
}

type postFn func(ctx context.Context) error

func postToAll(ctx context.Context, fns []postFn) error {
	respCh := make(chan error, len(fns))
	for _, fn := range fns {
		go func(fn postFn) {
			err := fn(ctx)
			if err != nil {
				respCh <- err
				return
			}
			respCh <- nil
		}(fn)
	}

	errCnt := 0
	var lastError error
	for i := 0; i < len(fns); i++ {
		respErr := <-respCh
		if respErr != nil {
			lastError = respErr
			errCnt++
		}
	}
	// at least one success
	if errCnt < len(fns) {
		return nil
	}
	return lastError
}
