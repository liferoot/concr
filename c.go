package concr

import (
	"sync/atomic"
	"time"
	"unsafe"
)

type C struct {
	state uint64
	idle  unsafe.Pointer
}

func (c *C) Inc() uint32 {
	n := atomic.AddUint64(&c.state, 1)
	if n&valueMask == 1 {
		for !atomic.CompareAndSwapUint64(&c.state, n, n|waitLock) {
		}
	}
	return uint32(n)
}

func (c *C) Dec() uint32 {
	return uint32(atomic.AddUint64(&c.state, ^uint64(0)))
}

func (c *C) Get() (uint32, uint32) {
	n := atomic.LoadUint64(&c.state)
	return uint32(n & valueMask), uint32(n & limitMask >> 31)
}

func (c *C) Set(m uint32) {
	var n uint64

	for {
		n = atomic.LoadUint64(&c.state)
		if atomic.CompareAndSwapUint64(&c.state, n, n&(valueMask|waitLock)|uint64(m)<<31) {
			return
		}
	}
}

func (c *C) Within() bool {
	n := atomic.LoadUint64(&c.state)
	return n&valueMask < n&limitMask>>31
}

func (c *C) Reached() bool {
	n := atomic.LoadUint64(&c.state)
	v := n & valueMask
	return v > 0 && v == n&limitMask>>31
}

func (c *C) Exceeded() bool {
	n := atomic.LoadUint64(&c.state)
	v := n & valueMask
	return v > 0 && v > n&limitMask>>31
}

func (c *C) Wait() {
	var v uint64

	for {
		c.Idle()

		v = atomic.LoadUint64(&c.state)
		if v&waitLock == 0 {
			continue
		}
		if v&valueMask == 0 {
			break
		}
	}
	for !atomic.CompareAndSwapUint64(&c.state, v, v|waitLock^waitLock) {
	}
}

func (c *C) Idle() {
	p := atomic.LoadPointer(&c.idle)
	if p == nil {
		p = unsafe.Pointer(&defaultIdle)
		atomic.StorePointer(&c.idle, p)
	}
	(*(*func())(p))()
}

func (c *C) SetIdle(f func()) {
	if f == nil {
		f = defaultIdle
	}
	atomic.StorePointer(&c.idle, unsafe.Pointer(&f))
}

var defaultIdle = func() { time.Sleep(time.Duration(100 * time.Millisecond)) }

const (
	MAX = uint64(1<<30 - 1)

	// 1001111111111111111111111111111110111111111111111111111111111111
	valueMask = MAX
	limitMask = uint64(valueMask << 31)
	waitLock  = uint64(1 << 63)
)
