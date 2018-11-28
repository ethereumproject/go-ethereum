// Copyright 2015 The go-ethereum Authors
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

package miner

import (
	"sync"
	"time"

	"sync/atomic"

	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/core/types"
	"github.com/ethereumproject/go-ethereum/logger"
	"github.com/ethereumproject/go-ethereum/logger/glog"
	"github.com/ethereumproject/go-ethereum/pow"
)

type AutoAgent struct {
	mu sync.Mutex

	workCh        chan *Work
	quit          chan struct{}
	quitCurrentOp chan struct{}
	returnCh      chan<- *Result

	index int
	pow   pow.PoW

	isMining int32 // isMining indicates whether the agent is currently mining
}

func NewAutoAgent(index int) *AutoAgent {
	miner := &AutoAgent{
		index: index,
	}

	return miner
}

func (self *AutoAgent) Work() chan<- *Work            { return self.workCh }
func (self *AutoAgent) Pow() pow.PoW                  { return self.pow }
func (self *AutoAgent) SetReturnCh(ch chan<- *Result) { self.returnCh = ch }

func (self *AutoAgent) Stop() {
	self.mu.Lock()
	defer self.mu.Unlock()

	close(self.quit)
}

func (self *AutoAgent) Start() {
	self.mu.Lock()
	defer self.mu.Unlock()

	if !atomic.CompareAndSwapInt32(&self.isMining, 0, 1) {
		return // agent already started
	}

	self.quit = make(chan struct{})
	// creating current op ch makes sure we're not closing a nil ch
	// later on
	self.workCh = make(chan *Work, 1)

	go self.update()
}

func (self *AutoAgent) update() {
out:
	for {
		select {
		case work := <-self.workCh:
			self.mu.Lock()
			if self.quitCurrentOp != nil {
				close(self.quitCurrentOp)
			}
			self.quitCurrentOp = make(chan struct{})
			go self.mine(work, self.quitCurrentOp)
			self.mu.Unlock()
		case <-self.quit:
			self.mu.Lock()
			if self.quitCurrentOp != nil {
				close(self.quitCurrentOp)
				self.quitCurrentOp = nil
			}
			self.mu.Unlock()
			break out
		}
	}

done:
	// Empty work channel
	for {
		select {
		case <-self.workCh:
		default:
			close(self.workCh)
			break done
		}
	}

	atomic.StoreInt32(&self.isMining, 0)
}

func (self *AutoAgent) mine(work *Work, stop <-chan struct{}) {
	glog.V(logger.Debug).Infof("(re)started agent[%d]. mining...\n", self.index)

	// Mine
	// nonce, mixDigest := self.pow.Search(work.Block, stop, self.index)
	nonce := work.Block.NumberU64() + 1
	mixDigest, _ := time.Now().MarshalBinary()
	if nonce != 0 {
		block := work.Block.WithMiningResult(nonce, common.BytesToHash(mixDigest))
		self.returnCh <- &Result{work, block}
	} else {
		self.returnCh <- nil
	}
}

func (self *AutoAgent) GetHashRate() int64 {
	return self.pow.GetHashrate()
}

func (self *AutoAgent) Win(work *Work) *types.Block {
	nonce := uint64(time.Now().UnixNano())
	mixDigest := common.BytesToHash(nil)
	block := work.Block.WithMiningResult(nonce, mixDigest)
	return block
}
