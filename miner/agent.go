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

	"sync/atomic"

	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/core/types"
	"github.com/ethereumproject/go-ethereum/logger"
	"github.com/ethereumproject/go-ethereum/logger/glog"
	"github.com/ethereumproject/go-ethereum/pow"
)

type CpuAgent struct {
	mu sync.Mutex

	workCh        chan *Work
	quit          chan struct{}
	quitCurrentOp chan struct{}
	returnCh      chan<- *Result

	index int
	pow   pow.PoW

	isMining int32 // isMining indicates whether the agent is currently mining
}

func NewCpuAgent(index int, pow pow.PoW) *CpuAgent {
	miner := &CpuAgent{
		pow:   pow,
		index: index,
	}

	return miner
}

func (aa *CpuAgent) Work() chan<- *Work            { return aa.workCh }
func (aa *CpuAgent) Pow() pow.PoW                  { return aa.pow }
func (aa *CpuAgent) SetReturnCh(ch chan<- *Result) { aa.returnCh = ch }

func (aa *CpuAgent) Stop() {
	aa.mu.Lock()
	defer aa.mu.Unlock()

	close(aa.quit)
}

func (aa *CpuAgent) Start() {
	aa.mu.Lock()
	defer aa.mu.Unlock()

	if !atomic.CompareAndSwapInt32(&aa.isMining, 0, 1) {
		return // agent already started
	}

	aa.quit = make(chan struct{})
	// creating current op ch makes sure we're not closing a nil ch
	// later on
	aa.workCh = make(chan *Work, 1)

	go aa.update()
}

func (aa *CpuAgent) update() {
out:
	for {
		select {
		case work := <-aa.workCh:
			aa.mu.Lock()
			if aa.quitCurrentOp != nil {
				close(aa.quitCurrentOp)
			}
			aa.quitCurrentOp = make(chan struct{})
			go aa.mine(work, aa.quitCurrentOp)
			aa.mu.Unlock()
		case <-aa.quit:
			aa.mu.Lock()
			if aa.quitCurrentOp != nil {
				close(aa.quitCurrentOp)
				aa.quitCurrentOp = nil
			}
			aa.mu.Unlock()
			break out
		}
	}

done:
	// Empty work channel
	for {
		select {
		case <-aa.workCh:
		default:
			close(aa.workCh)
			break done
		}
	}

	atomic.StoreInt32(&aa.isMining, 0)
}

func (aa *CpuAgent) mine(work *Work, stop <-chan struct{}) {
	glog.V(logger.Debug).Infof("(re)started agent[%d]. mining...\n", aa.index)

	// Mine
	nonce, mixDigest := aa.pow.Search(work.Block, stop, aa.index)
	if nonce != 0 {
		block := work.Block.WithMiningResult(nonce, common.BytesToHash(mixDigest))
		aa.returnCh <- &Result{work, block}
	} else {
		aa.returnCh <- nil
	}
}

func (aa *CpuAgent) GetHashRate() int64 {
	return aa.pow.GetHashrate()
}

func (aa *CpuAgent) Win(work *Work) *types.Block {
	panic("satisfies automining method")
}
