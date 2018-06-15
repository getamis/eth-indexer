// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package state

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

type DumpAccount struct {
	Balance  string            `json:"balance"`
	Nonce    uint64            `json:"nonce"`
	Root     string            `json:"root"`
	CodeHash string            `json:"codeHash"`
	Code     string            `json:"code"`
	Storage  map[string]string `json:"storage"`
}

type Dump struct {
	Root     string                 `json:"root"`
	Accounts map[string]DumpAccount `json:"accounts"`
}

func (self *StateDB) RawDump() Dump {
	dump := Dump{
		Root:     fmt.Sprintf("%x", self.trie.Hash()),
		Accounts: make(map[string]DumpAccount),
	}

	it := trie.NewIterator(self.trie.NodeIterator(nil))
	for it.Next() {
		addr := self.trie.GetKey(it.Key)
		var data Account
		if err := rlp.DecodeBytes(it.Value, &data); err != nil {
			panic(err)
		}

		obj := newObject(nil, common.BytesToAddress(addr), data)
		account := DumpAccount{
			Balance:  data.Balance.String(),
			Nonce:    data.Nonce,
			Root:     common.Bytes2Hex(data.Root[:]),
			CodeHash: common.Bytes2Hex(data.CodeHash),
			Code:     common.Bytes2Hex(obj.Code(self.db)),
			Storage:  make(map[string]string),
		}
		storageIt := trie.NewIterator(obj.getTrie(self.db).NodeIterator(nil))
		for storageIt.Next() {
			account.Storage[common.Bytes2Hex(self.trie.GetKey(storageIt.Key))] = common.Bytes2Hex(storageIt.Value)
		}
		dump.Accounts[common.Bytes2Hex(addr)] = account
	}
	return dump
}

func (self *StateDB) Dump() []byte {
	json, err := json.MarshalIndent(self.RawDump(), "", "    ")
	if err != nil {
		fmt.Println("dump err", err)
	}

	return json
}

type DirtyDump struct {
	Root     string                      `json:"root"`
	Accounts map[string]DirtyDumpAccount `json:"accounts"`
}

func newDirtyDump(trie Trie) *DirtyDump {
	return &DirtyDump{
		Root:     fmt.Sprintf("%x", trie.Hash()),
		Accounts: make(map[string]DirtyDumpAccount),
	}
}

func (d *DirtyDump) Copy() *DirtyDump {
	cpy := &DirtyDump{
		Root:     d.Root,
		Accounts: make(map[string]DirtyDumpAccount),
	}
	for account, dirty := range d.Accounts {
		storage := make(map[string]string)
		for key, val := range dirty.Storage {
			storage[key] = val
		}
		cpy.Accounts[account] = DirtyDumpAccount{
			Balance: dirty.Balance,
			Storage: storage,
		}
	}
	return cpy
}

// EncodeRLP implements rlp.Encoder.
func (d *DirtyDump) EncodeRLP(w io.Writer) error {
	dirtyDumpRLP := dirtyDumpRLP{
		Root:     d.Root,
		Accounts: make([]dirtyDumpAccountRLP, 0),
	}
	for account, dirtyDumpAccount := range d.Accounts {
		accountRLP := dirtyDumpAccountRLP{
			Account: account,
			Balance: dirtyDumpAccount.Balance,
		}
		if len(dirtyDumpAccount.Storage) > 0 {
			accountRLP.Storage = make([]dirtyStorageRLP, 0)
			for key, val := range dirtyDumpAccount.Storage {
				accountRLP.Storage = append(accountRLP.Storage, dirtyStorageRLP{Key: key, Value: val})
			}
		}
		dirtyDumpRLP.Accounts = append(dirtyDumpRLP.Accounts, accountRLP)
	}
	return rlp.Encode(w, &dirtyDumpRLP)
}

// DecodeRLP implements rlp.Decoder.
func (d *DirtyDump) DecodeRLP(s *rlp.Stream) error {
	var dec dirtyDumpRLP
	if err := s.Decode(&dec); err != nil {
		return err
	}
	d.Root = dec.Root
	d.Accounts = make(map[string]DirtyDumpAccount)
	for _, accountRLP := range dec.Accounts {
		account := DirtyDumpAccount{
			Balance: accountRLP.Balance,
		}
		if len(accountRLP.Storage) > 0 {
			account.Storage = make(map[string]string)
			for _, storageRLP := range accountRLP.Storage {
				account.Storage[storageRLP.Key] = storageRLP.Value
			}
		}
		d.Accounts[accountRLP.Account] = account
	}
	return nil
}

// DirtyDumpAccount records the changed balance and storage for an account.
type DirtyDumpAccount struct {
	Balance string            `json:"balance,omitempty"`
	Storage map[string]string `json:"storage,omitempty"`
}

// DumpDirty return the dirty storage diff.
func (self *StateDB) DumpDirty() *DirtyDump {
	return self.dirtyDump
}

// dumpDirtySnapshot dumps the balances and dirty storage for each dirty accounts in current state.
func (self *StateDB) dumpDirtySnapshot() {
	for addr, change := range calcDirties(self.journal.entries) {
		account, exist := self.dirtyDump.Accounts[common.Bytes2Hex(addr.Bytes())]
		if !exist {
			account = DirtyDumpAccount{Storage: make(map[string]string)}
		}

		if change.balanceChange > 0 {
			account.Balance = self.GetBalance(addr).String()
		}

		if len(change.storageChange) > 0 {
			for key := range change.storageChange {
				value := self.GetState(addr, key)
				account.Storage[common.Bytes2Hex(key.Bytes())] = common.Bytes2Hex(value.Bytes())
			}
		}
		self.dirtyDump.Accounts[common.Bytes2Hex(addr.Bytes())] = account
	}
}

// dirtyDiff records how many balance changes and changed keys of storage for one account.
type dirtyDiff struct {
	balanceChange int
	storageChange map[common.Hash]struct{}
}

// calcDirties calculates balance change and storage change by account.
func calcDirties(dirtyEntry []journalEntry) map[common.Address]*dirtyDiff {
	dirties := make(map[common.Address]*dirtyDiff)
	for _, entry := range dirtyEntry {
		if addr := entry.dirtied(); addr != nil {
			if dirties[*addr] == nil {
				dirties[*addr] = &dirtyDiff{storageChange: make(map[common.Hash]struct{})}
			}
			switch v := entry.(type) {
			case balanceChange:
				dirties[*addr].balanceChange++
			case storageChange:
				dirties[*addr].storageChange[v.key] = struct{}{}
			}
		}
	}
	return dirties
}

type dirtyDumpRLP struct {
	Root     string
	Accounts []dirtyDumpAccountRLP
}

type dirtyDumpAccountRLP struct {
	Account string
	Balance string
	Storage []dirtyStorageRLP
}

type dirtyStorageRLP struct {
	Key   string
	Value string
}
