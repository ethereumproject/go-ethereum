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
	"bufio"
	"bytes"
	"compress/zlib"
	"encoding/json"
	"io"
	"sort"
	"sync"

	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/rlp"
)

type DumpAccount struct {
	Balance  string            `json:"balance"`
	Nonce    uint64            `json:"nonce"`
	Root     string            `json:"root"`
	CodeHash string            `json:"codeHash"`
	Code     string            `json:"code"`
	Storage  map[string]string `json:"storage"`
}

type Dumps []Dump

type Dump struct {
	Root     string                 `json:"root"`
	Accounts map[string]DumpAccount `json:"accounts"`
}

func lookupAddress(addr common.Address, addresses []common.Address) bool {
	for _, add := range addresses {
		if add.Hex() == addr.Hex() {
			return true
		}
	}
	return false
}

func (self *StateDB) RawDump(addresses []common.Address) Dump {

	dump := Dump{
		Root:     common.Bytes2Hex(self.trie.Root()),
		Accounts: make(map[string]DumpAccount),
	}

	it := self.trie.Iterator()
	for it.Next() {
		addr := self.trie.GetKey(it.Key)
		addrA := common.BytesToAddress(addr)

		if addresses != nil && len(addresses) > 0 {
			// check if address existing in argued addresses (lookup)
			// if it's not one we're looking for, continue
			if !lookupAddress(addrA, addresses) {
				continue
			}
		}

		var data Account
		if err := rlp.DecodeBytes(it.Value, &data); err != nil {
			panic(err)
		}

		obj := newObject(nil, addrA, data, nil)
		account := DumpAccount{
			Balance:  data.Balance.String(),
			Nonce:    data.Nonce,
			Root:     common.Bytes2Hex(data.Root[:]),
			CodeHash: common.Bytes2Hex(data.CodeHash),
			Code:     common.Bytes2Hex(obj.Code(self.db)),
			Storage:  make(map[string]string),
		}
		storageIt := obj.getTrie(self.db).Iterator()
		for storageIt.Next() {
			account.Storage[common.Bytes2Hex(self.trie.GetKey(storageIt.Key))] = common.Bytes2Hex(storageIt.Value)
		}
		dump.Accounts[common.Bytes2Hex(addr)] = account
	}
	return dump
}

const ZipperBlockLength = 1 * 1024 * 1024
const ZipperPieceLength = 64 * 1024

type Zipper struct {
	MemBrk int
	Mem    []byte
	Bf     bytes.Buffer
}

type AddressedRawAccount struct {
	DumpAccount
	Addr string
}

type EncodedAccount struct {
	Addr  string
	Json  []byte
	Error error
}

func (self *Zipper) ZipBytes(data []byte) (result []byte, err error) {
	self.Bf.Reset()
	wr, err := zlib.NewWriterLevel(&self.Bf, zlib.DefaultCompression)
	if err != nil {
		panic(err)
	}
	if _, err := wr.Write(data); err != nil {
		panic(err)
	}
	wr.Close()
	n := self.Bf.Len()
	if n == 0 {
		n = 1
	}
	if n > ZipperPieceLength {
		result = self.Bf.Bytes()
		self.Bf = bytes.Buffer{}
	} else {
		if n+self.MemBrk > ZipperBlockLength || self.Mem == nil {
			self.Mem = make([]byte, ZipperBlockLength)
			self.MemBrk = 0
		}
		result = self.Mem[self.MemBrk : self.MemBrk+n]
		self.MemBrk = self.MemBrk + n
	}
	copy(result, self.Bf.Bytes())
	return
}

func (self *Zipper) UnZipBytes(data []byte) (result []byte, err error) {
	var bf bytes.Buffer
	r, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return
	}
	io.Copy(&bf, r)
	r.Close()
	result = bf.Bytes()
	return
}

func iterator(sdb *StateDB, addresses []common.Address, c chan *AddressedRawAccount) {
	it := sdb.trie.Iterator()
	for it.Next() {
		addr := sdb.trie.GetKey(it.Key)
		addrA := common.BytesToAddress(addr)

		if addresses != nil && len(addresses) > 0 {
			// check if address existing in argued addresses (lookup)
			// if it's not one we're looking for, continue
			if !lookupAddress(addrA, addresses) {
				continue
			}
		}

		var data Account
		if err := rlp.DecodeBytes(it.Value, &data); err != nil {
			panic(err)
		}

		obj := newObject(nil, addrA, data, nil)
		account := AddressedRawAccount{
			DumpAccount: DumpAccount{
				Balance:  data.Balance.String(),
				Nonce:    data.Nonce,
				Root:     common.Bytes2Hex(data.Root[:]),
				CodeHash: common.Bytes2Hex(data.CodeHash),
				Code:     common.Bytes2Hex(obj.Code(sdb.db)),
				Storage:  make(map[string]string)},
			Addr: common.Bytes2Hex(addr),
		}
		storageIt := obj.getTrie(sdb.db).Iterator()
		for storageIt.Next() {
			account.Storage[common.Bytes2Hex(sdb.trie.GetKey(storageIt.Key))] = common.Bytes2Hex(storageIt.Value)
		}
		c <- &account
	}
	close(c)
}

func compressor(c chan *AddressedRawAccount, cN chan EncodedAccount, wg *sync.WaitGroup) {
	var zipper Zipper
	defer wg.Done()
	for {
		select {
		case account, ok := <-c:
			if !ok {
				return
			}
			if val, err := json.Marshal(account.DumpAccount); err != nil {
				cN <- EncodedAccount{"", nil, err}
				return
			} else {
				data, err := zipper.ZipBytes(val)
				cN <- EncodedAccount{account.Addr, data, err}
				if err != nil {
					return
				}
			}
		}
	}
}

func encoder(c chan *AddressedRawAccount, cN chan EncodedAccount, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case account, ok := <-c:
			if !ok {
				return
			}
			if val, err := json.Marshal(account.DumpAccount); err != nil {
				cN <- EncodedAccount{"", nil, err}
				return
			} else {
				cN <- EncodedAccount{account.Addr, val, nil}
				if err != nil {
					return
				}
			}
		}
	}
}

func (self *StateDB) LoadEncodedAccounts(addresses []common.Address) (accounts map[string][]byte, err error) {

	accounts = make(map[string][]byte)

	var wg sync.WaitGroup
	c1 := make(chan *AddressedRawAccount, 10)
	c2 := make(chan EncodedAccount, 10)
	c3 := make(chan error)

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go compressor(c1, c2, &wg)
	}

	go iterator(self, addresses, c1)

	go func() {
		for {
			acc, ok := <-c2
			if !ok {
				c3 <- nil
				return
			}
			if acc.Error != nil {
				c3 <- err
				return
			}
			accounts[acc.Addr] = acc.Json
		}
	}()

	wg.Wait()
	close(c2)
	err = <-c3
	return
}

func writer(root string, zipped bool, prefix string, indent string, out io.Writer) func(chan EncodedAccount, chan error) {

	return func(c chan EncodedAccount, cError chan error) {
		var err error
		defer func() {
			cError <- err
		}()

		indent2 := prefix + indent
		indent3 := indent2 + indent

		var (
			js []byte
			bf bytes.Buffer
		)

		wr := bufio.NewWriter(out)

		wr.WriteString(prefix)
		wr.WriteString("{\n")
		wr.WriteString(indent2)
		wr.WriteString("\"root\": ")

		js, err = json.Marshal(root)
		if err != nil {
			return
		}

		wr.Write(js)
		wr.WriteString(",\n")
		wr.WriteString(indent2)
		wr.WriteString("\"accounts\": {\n")
		nl := false

	loop:
		for {
			select {
			case acc, ok := <-c:

				if !ok {
					break loop
				}

				if acc.Error != nil {
					err = acc.Error
					return
				}

				if !nl {
					nl = true
				} else {
					wr.WriteString(",\n")
				}

				wr.WriteString(indent3)
				js, err = json.Marshal(acc.Addr)
				if err != nil {
					return
				}

				wr.Write(js)
				wr.WriteString(": ")
				bf.Reset()
				if zipped {
					var zipper Zipper
					var r []byte
					r, err = zipper.UnZipBytes(acc.Json)
					if err != nil {
						return
					}
					json.Indent(&bf, r, indent3, indent)
				} else {
					json.Indent(&bf, acc.Json, indent3, indent)
				}
				wr.Write(bf.Bytes())
			}
		}

		wr.WriteString("\n")
		wr.WriteString(indent2)
		wr.WriteString("}\n")
		wr.WriteString(prefix)
		wr.WriteString("}")
		wr.Flush()

		err = nil
		return
	}
}

func (self *StateDB) UnsortedRawDump(addresses []common.Address, fwr func(chan EncodedAccount, chan error)) (err error) {

	var wg sync.WaitGroup
	c1 := make(chan *AddressedRawAccount, 10)
	c2 := make(chan EncodedAccount, 10)
	c3 := make(chan error)
	go iterator(self, addresses, c1)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go encoder(c1, c2, &wg)
	}
	go fwr(c2, c3)
	wg.Wait()
	close(c2)
	err = <-c3
	return
}

func (self *StateDB) SortedDump(addresses []common.Address, prefix string, indent string, out io.Writer) (err error) {

	var accounts map[string][]byte

	accounts, err = self.LoadEncodedAccounts(addresses)
	if err != nil {
		return
	}

	fwr := writer(common.Bytes2Hex(self.trie.Root()), true, prefix, indent, out)

	keys := make([]string, 0, len(accounts))
	for k := range accounts {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	c1 := make(chan EncodedAccount, 1)
	c2 := make(chan error, 1)

	go fwr(c1, c2)

	go func() {
		for _, addr := range keys {
			data := accounts[addr]
			c1 <- EncodedAccount{addr, data, nil}
		}
		close(c1)
	}()

	err = <-c2
	return
}

func (self *StateDB) UnsortedDump(addresses []common.Address, prefix string, indent string, out io.Writer) (err error) {
	fwr := writer(common.Bytes2Hex(self.trie.Root()), false, prefix, indent, out)
	return self.UnsortedRawDump(addresses, fwr)
}

func (self *StateDB) Dump(addresses []common.Address) []byte {
	var bf bytes.Buffer
	err := self.SortedDump(addresses, "", "    ", &bf)
	if err != nil {
		return nil
	}
	return bf.Bytes()
}
