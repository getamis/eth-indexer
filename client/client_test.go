package client

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

func ExampleUncleByNumberAndPosition() {
	endpoint := "ws://127.0.0.1:8546"
	c, err := NewClient(endpoint)
	if err != nil {
		fmt.Println("Failed to dial ethereum rpc", "endpoint", endpoint, "err", err)
		return
	}

	blockHash := common.HexToHash("0x770020d5a1dac6f54b13913849fb4b07ccd2daa719e21af37e4b46e81b33c627")
	position := 0

	h, err := c.UncleByBlockHashAndPosition(context.Background(), blockHash, uint(position))
	if err != nil {
		fmt.Println("Failed to get uncle", "number", endpoint, "err", err)
		return
	}

	//// Output: 5936784
	//// 0x22714689efa7bbd55afc8145a4faf3592231a250acc77d122cdde3bb4c12b4b9
	//// 7973852
	//// 7992185
	fmt.Println(h.Number.String())
	fmt.Println(h.Hash().Hex())
	fmt.Println(h.GasUsed)
	fmt.Println(h.GasLimit)
	fmt.Println(h.Root.Hex())
}

func ExampleUnclesByBlockHashWith2Uncles() {
	endpoint := "ws://127.0.0.1:8546"
	c, err := NewClient(endpoint)
	if err != nil {
		fmt.Println("Failed to dial ethereum rpc", "endpoint", endpoint, "err", err)
		return
	}

	blockHash := common.HexToHash("0x770020d5a1dac6f54b13913849fb4b07ccd2daa719e21af37e4b46e81b33c627")
	b, err := c.BlockByHash(context.Background(), blockHash)
	if err != nil {
		fmt.Println("Failed to get block", "target hash", blockHash, "err", err)
		return
	}

	hs, err := c.UnclesByBlockHash(context.Background(), b.Hash())
	if err != nil {
		fmt.Println("Failed to get uncle", "number", endpoint, "err", err)
		return
	}

	//// Output: 5936784
	//// 5936786
	//// 0x22714689efa7bbd55afc8145a4faf3592231a250acc77d122cdde3bb4c12b4b9
	//// 0xa6590d9bccdce928c2e77efe74ab68c2ab5014b997aa9e7aaad7fad512699163
	fmt.Println(hs[0].Number.String())
	fmt.Println(hs[1].Number.String())
	fmt.Println(hs[0].Hash().Hex())
	fmt.Println(hs[1].Hash().Hex())
}

func ExampleUnclesByBlockHashWithNoUncle() {
	endpoint := "ws://127.0.0.1:8546"
	c, err := NewClient(endpoint)
	if err != nil {
		fmt.Println("Failed to dial ethereum rpc", "endpoint", endpoint, "err", err)
		return
	}

	blockHash := common.HexToHash("0xdd0de7f9930ce464d4606be8f6679eb3f55d5022022c11014aad43ea911faa7f")
	b, err := c.BlockByHash(context.Background(), blockHash)
	if err != nil {
		fmt.Println("Failed to get block", "target hash", blockHash, "err", err)
		return
	}

	hs, err := c.UnclesByBlockHash(context.Background(), b.Hash())
	if err != nil {
		fmt.Println("Failed to get uncle", "number", endpoint, "err", err)
		return
	}

	//// Output: 0
	fmt.Println(len(hs))
}
