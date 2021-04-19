// Copyright 2019 The go-ays Authors
// This file is part of the go-ays library.
//
// The go-ays library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ays library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ays library. If not, see <http://www.gnu.org/licenses/>.

package server

import (
	"math/rand"
	"reflect"
	"testing"
	"time"

	"github.com/ays/go-ays/common/mclock"
	"github.com/ays/go-ays/aysdb/memorydb"
	"github.com/ays/go-ays/les/utils"
	"github.com/ays/go-ays/p2p/enode"
	"github.com/ays/go-ays/p2p/enr"
	"github.com/ays/go-ays/p2p/nodestate"
)

var (
	testFlag     = testSetup.NewFlag("testFlag")
	connAddrFlag = testSetup.NewField("connAddr", reflect.TypeOf(""))
	btTestSetup  = NewBalanceTrackerSetup(testSetup)
)

func init() {
	btTestSetup.Connect(connAddrFlag, ppTestSetup.CapacityField)
}

type zeroExpirer struct{}

func (z zeroExpirer) SetRate(now mclock.AbsTime, rate float64)                 {}
func (z zeroExpirer) SetLogOffset(now mclock.AbsTime, logOffset utils.Fixed64) {}
func (z zeroExpirer) LogOffset(now mclock.AbsTime) utils.Fixed64               { return 0 }

type balanceTestSetup struct {
	clock *mclock.Simulated
	ns    *nodestate.NodeStateMachine
	bt    *BalanceTracker
}

func newBalanceTestSetup() *balanceTestSetup {
	clock := &mclock.Simulated{}
	ns := nodestate.NewNodeStateMachine(nil, nil, clock, testSetup)
	db := memorydb.New()
	bt := NewBalanceTracker(ns, btTestSetup, db, clock, zeroExpirer{}, zeroExpirer{})
	ns.Start()
	return &balanceTestSetup{
		clock: clock,
		ns:    ns,
		bt:    bt,
	}
}

func (b *balanceTestSetup) newNode(capacity uint64) *NodeBalance {
	node := enode.SignNull(&enr.Record{}, enode.ID{})
	b.ns.SetState(node, testFlag, nodestate.Flags{}, 0)
	b.ns.SetField(node, btTestSetup.connAddressField, "")
	b.ns.SetField(node, ppTestSetup.CapacityField, capacity)
	n, _ := b.ns.GetField(node, btTestSetup.BalanceField).(*NodeBalance)
	return n
}

func (b *balanceTestSetup) stop() {
	b.bt.Stop()
	b.ns.Stop()
}

func TestAddBalance(t *testing.T) {
	b := newBalanceTestSetup()
	defer b.stop()

	node := b.newNode(1000)
	var inputs = []struct {
		delta     int64
		expect    [2]uint64
		total     uint64
		expectErr bool
	}{
		{100, [2]uint64{0, 100}, 100, false},
		{-100, [2]uint64{100, 0}, 0, false},
		{-100, [2]uint64{0, 0}, 0, false},
		{1, [2]uint64{0, 1}, 1, false},
		{maxBalance, [2]uint64{0, 0}, 0, true},
	}
	for _, i := range inputs {
		old, new, err := node.AddBalance(i.delta)
		if i.expectErr {
			if err == nil {
				t.Fatalf("Expect get error but nil")
			}
			continue
		} else if err != nil {
			t.Fatalf("Expect get no error but %v", err)
		}
		if old != i.expect[0] || new != i.expect[1] {
			t.Fatalf("Positive balance mismatch, got %v -> %v", old, new)
		}
		if b.bt.TotalTokenAmount() != i.total {
			t.Fatalf("Total positive balance mismatch, want %v, got %v", i.total, b.bt.TotalTokenAmount())
		}
	}
}

func TestSetBalance(t *testing.T) {
	b := newBalanceTestSetup()
	defer b.stop()
	node := b.newNode(1000)

	var inputs = []struct {
		pos, neg uint64
	}{
		{1000, 0},
		{0, 1000},
		{1000, 1000},
	}

	for _, i := range inputs {
		node.SetBalance(i.pos, i.neg)
		pos, neg := node.GetBalance()
		if pos != i.pos {
			t.Fatalf("Positive balance mismatch, want %v, got %v", i.pos, pos)
		}
		if neg != i.neg {
			t.Fatalf("Negative balance mismatch, want %v, got %v", i.neg, neg)
		}
	}
}

func TestBalanceTimeCost(t *testing.T) {
	b := newBalanceTestSetup()
	defer b.stop()
	node := b.newNode(1000)

	b.ns.SetField(node.node, ppTestSetup.CapacityField, uint64(1))
	node.SetPriceFactors(PriceFactors{1, 0, 1}, PriceFactors{1, 0, 1})
	node.SetBalance(uint64(time.Minute), 0) // 1 minute time allowance

	var inputs = []struct {
		runTime time.Duration
		expPos  uint64
		expNeg  uint64
	}{
		{time.Second, uint64(time.Second * 59), 0},
		{0, uint64(time.Second * 59), 0},
		{time.Second * 59, 0, 0},
		{time.Second, 0, uint64(time.Second)},
	}
	for _, i := range inputs {
		b.clock.Run(i.runTime)
		if pos, _ := node.GetBalance(); pos != i.expPos {
			t.Fatalf("Positive balance mismatch, want %v, got %v", i.expPos, pos)
		}
		if _, neg := node.GetBalance(); neg != i.expNeg {
			t.Fatalf("Negative balance mismatch, want %v, got %v", i.expNeg, neg)
		}
	}

	node.SetBalance(uint64(time.Minute), 0) // Refill 1 minute time allowance
	for _, i := range inputs {
		b.clock.Run(i.runTime)
		if pos, _ := node.GetBalance(); pos != i.expPos {
			t.Fatalf("Positive balance mismatch, want %v, got %v", i.expPos, pos)
		}
		if _, neg := node.GetBalance(); neg != i.expNeg {
			t.Fatalf("Negative balance mismatch, want %v, got %v", i.expNeg, neg)
		}
	}
}

func TestBalanceReqCost(t *testing.T) {
	b := newBalanceTestSetup()
	defer b.stop()
	node := b.newNode(1000)
	node.SetPriceFactors(PriceFactors{1, 0, 1}, PriceFactors{1, 0, 1})

	b.ns.SetField(node.node, ppTestSetup.CapacityField, uint64(1))
	node.SetBalance(uint64(time.Minute), 0) // 1 minute time serving time allowance
	var inputs = []struct {
		reqCost uint64
		expPos  uint64
		expNeg  uint64
	}{
		{uint64(time.Second), uint64(time.Second * 59), 0},
		{0, uint64(time.Second * 59), 0},
		{uint64(time.Second * 59), 0, 0},
		{uint64(time.Second), 0, uint64(time.Second)},
	}
	for _, i := range inputs {
		node.RequestServed(i.reqCost)
		if pos, _ := node.GetBalance(); pos != i.expPos {
			t.Fatalf("Positive balance mismatch, want %v, got %v", i.expPos, pos)
		}
		if _, neg := node.GetBalance(); neg != i.expNeg {
			t.Fatalf("Negative balance mismatch, want %v, got %v", i.expNeg, neg)
		}
	}
}

func TestBalanceToPriority(t *testing.T) {
	b := newBalanceTestSetup()
	defer b.stop()
	node := b.newNode(1000)
	node.SetPriceFactors(PriceFactors{1, 0, 1}, PriceFactors{1, 0, 1})

	var inputs = []struct {
		pos      uint64
		neg      uint64
		priority int64
	}{
		{1000, 0, 1},
		{2000, 0, 2}, // Higher balance, higher priority value
		{0, 0, 0},
		{0, 1000, -1000},
	}
	for _, i := range inputs {
		node.SetBalance(i.pos, i.neg)
		priority := node.Priority(b.clock.Now(), 1000)
		if priority != i.priority {
			t.Fatalf("Priority mismatch, want %v, got %v", i.priority, priority)
		}
	}
}

func TestEstimatedPriority(t *testing.T) {
	b := newBalanceTestSetup()
	defer b.stop()
	node := b.newNode(1000000000)
	node.SetPriceFactors(PriceFactors{1, 0, 1}, PriceFactors{1, 0, 1})

	b.ns.SetField(node.node, ppTestSetup.CapacityField, uint64(1))
	node.SetBalance(uint64(time.Minute), 0)
	var inputs = []struct {
		runTime    time.Duration // time cost
		futureTime time.Duration // diff of future time
		reqCost    uint64        // single request cost
		priority   int64         // expected estimated priority
	}{
		{time.Second, time.Second, 0, 58},
		{0, time.Second, 0, 58},

		// 2 seconds time cost, 1 second estimated time cost, 10^9 request cost,
		// 10^9 estimated request cost per second.
		{time.Second, time.Second, 1000000000, 55},

		// 3 seconds time cost, 3 second estimated time cost, 10^9*2 request cost,
		// 4*10^9 estimated request cost.
		{time.Second, 3 * time.Second, 1000000000, 48},

		// All positive balance is used up
		{time.Second * 55, 0, 0, 0},

		// 1 minute estimated time cost, 4/58 * 10^9 estimated request cost per sec.
		{0, time.Minute, 0, -int64(time.Minute) - int64(time.Second)*120/29},
	}
	for _, i := range inputs {
		b.clock.Run(i.runTime)
		node.RequestServed(i.reqCost)
		priority := node.EstMinPriority(b.clock.Now()+mclock.AbsTime(i.futureTime), 1000000000, false)
		if priority != i.priority {
			t.Fatalf("Estimated priority mismatch, want %v, got %v", i.priority, priority)
		}
	}
}

func TestPosBalanceMissing(t *testing.T) {
	b := newBalanceTestSetup()
	defer b.stop()
	node := b.newNode(1000)
	node.SetPriceFactors(PriceFactors{1, 0, 1}, PriceFactors{1, 0, 1})

	b.ns.SetField(node.node, ppTestSetup.CapacityField, uint64(1))
	var inputs = []struct {
		pos, neg uint64
		priority int64
		cap      uint64
		after    time.Duration
		expect   uint64
	}{
		{uint64(time.Second * 2), 0, 0, 1, time.Second, 0},
		{uint64(time.Second * 2), 0, 0, 1, 2 * time.Second, 1},
		{uint64(time.Second * 2), 0, int64(time.Second), 1, 2 * time.Second, uint64(time.Second) + 1},
		{0, 0, int64(time.Second), 1, time.Second, uint64(2*time.Second) + 1},
		{0, 0, -int64(time.Second), 1, time.Second, 1},
	}
	for _, i := range inputs {
		node.SetBalance(i.pos, i.neg)
		got := node.PosBalanceMissing(i.priority, i.cap, i.after)
		if got != i.expect {
			t.Fatalf("Missing budget mismatch, want %v, got %v", i.expect, got)
		}
	}
}

func TestPostiveBalanceCounting(t *testing.T) {
	b := newBalanceTestSetup()
	defer b.stop()

	var nodes []*NodeBalance
	for i := 0; i < 100; i += 1 {
		node := b.newNode(1000000)
		node.SetPriceFactors(PriceFactors{1, 0, 1}, PriceFactors{1, 0, 1})
		nodes = append(nodes, node)
	}

	// Allocate service token
	var sum uint64
	for i := 0; i < 100; i += 1 {
		amount := int64(rand.Intn(100) + 100)
		nodes[i].AddBalance(amount)
		sum += uint64(amount)
	}
	if b.bt.TotalTokenAmount() != sum {
		t.Fatalf("Invalid token amount")
	}

	// Change client status
	for i := 0; i < 100; i += 1 {
		if rand.Intn(2) == 0 {
			b.ns.SetField(nodes[i].node, ppTestSetup.CapacityField, uint64(1))
		}
	}
	if b.bt.TotalTokenAmount() != sum {
		t.Fatalf("Invalid token amount")
	}
	for i := 0; i < 100; i += 1 {
		if rand.Intn(2) == 0 {
			b.ns.SetField(nodes[i].node, ppTestSetup.CapacityField, uint64(1))
		}
	}
	if b.bt.TotalTokenAmount() != sum {
		t.Fatalf("Invalid token amount")
	}
}

func TestCallbackChecking(t *testing.T) {
	b := newBalanceTestSetup()
	defer b.stop()
	node := b.newNode(1000000)
	node.SetPriceFactors(PriceFactors{1, 0, 1}, PriceFactors{1, 0, 1})

	var inputs = []struct {
		priority int64
		expDiff  time.Duration
	}{
		{500, time.Millisecond * 500},
		{0, time.Second},
		{-int64(time.Second), 2 * time.Second},
	}
	node.SetBalance(uint64(time.Second), 0)
	for _, i := range inputs {
		diff, _ := node.timeUntil(i.priority)
		if diff != i.expDiff {
			t.Fatalf("Time difference mismatch, want %v, got %v", i.expDiff, diff)
		}
	}
}

func TestCallback(t *testing.T) {
	b := newBalanceTestSetup()
	defer b.stop()
	node := b.newNode(1000)
	node.SetPriceFactors(PriceFactors{1, 0, 1}, PriceFactors{1, 0, 1})
	b.ns.SetField(node.node, ppTestSetup.CapacityField, uint64(1))

	callCh := make(chan struct{}, 1)
	node.SetBalance(uint64(time.Minute), 0)
	node.addCallback(balanceCallbackZero, 0, func() { callCh <- struct{}{} })

	b.clock.Run(time.Minute)
	select {
	case <-callCh:
	case <-time.NewTimer(time.Second).C:
		t.Fatalf("Callback hasn't been called yet")
	}

	node.SetBalance(uint64(time.Minute), 0)
	node.addCallback(balanceCallbackZero, 0, func() { callCh <- struct{}{} })
	node.removeCallback(balanceCallbackZero)

	b.clock.Run(time.Minute)
	select {
	case <-callCh:
		t.Fatalf("Callback shouldn't be called")
	case <-time.NewTimer(time.Millisecond * 100).C:
	}
}