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

package main

import (
	"context"

	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
)

// Client defines typed wrappers for the Ethereum RPC API.
type Client struct {
	c *rpc.Client
}

func Dial(ctx context.Context, rawurl string) (*Client, error) {
	c, err := rpc.DialContext(ctx, rawurl)
	if err != nil {
		return nil, err
	}
	return NewClient(c), nil
}

// NewClient creates a client that uses the given RPC client.
func NewClient(c *rpc.Client) *Client {
	return &Client{c}
}

func (ec *Client) Close() {
	ec.c.Close()
}

func (ec *Client) Peers(ctx context.Context) ([]*p2p.PeerInfo, error) {
	var result []*p2p.PeerInfo
	err := ec.c.CallContext(ctx, &result, "admin_peers")
	return result, err
}

func (ec *Client) BatchAddPeer(ctx context.Context, urls []string) error {
	if len(urls) == 0 {
		return nil
	}
	// Construct batch requests
	method := "admin_addPeer"
	reqs := make([]rpc.BatchElem, len(urls))
	for i, url := range urls {
		reqs[i] = rpc.BatchElem{
			Method: method,
			Args:   []interface{}{url},
		}
	}
	// Batch calls
	err := ec.c.BatchCallContext(ctx, reqs)
	if err != nil {
		return err
	}
	// Ensure all requests are ok
	for _, req := range reqs {
		if req.Error != nil {
			return err
		}
	}
	return nil
}
