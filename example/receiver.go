package main

import (
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/liferoot/concr"
)

func main() {
	rx := new(Receiver)
	sig := make(chan os.Signal, 1)
	pid := os.Getpid()

	signal.Notify(sig, syscall.SIGUSR1, syscall.SIGUSR2)
	go func() {
		for {
			switch <-sig {
			case syscall.SIGUSR1:
				rx.c.Set(5)
			case syscall.SIGUSR2:
				rx.c.Set(10)
			}
			println()
			println()
		}
	}()
	fmt.Printf("usage:\nkill -s USR1 %d -- set concurrency to the 5.\n", pid)
	fmt.Printf("kill -s USR2 %d -- set concurrency to the 10.\n", pid)

	go rx.spawn()
	rx.c.Wait()
}

func payload() {
	rand.Seed(time.Now().Unix())
	time.Sleep(time.Duration(rand.Intn(800)+200) * time.Millisecond)
}

type Receiver struct {
	c concr.C
}

func (rx *Receiver) receive(id int) {
	rx.c.Inc()
	rx.echo(`start receive`, id)
	for {
		if rx.c.Exceeded() {
			rx.echo(`exceeded`, id)
			break
		}
		payload()
	}
	rx.c.Dec()
	rx.echo(`stop receive`, id)
}

func (rx *Receiver) spawn() {
	for i := 0; ; {
		if rx.c.Within() {
			go rx.receive(i)
			rx.echo(`spawn`, i)
			i++
		}
		rx.c.Idle()
	}
}

func (rx *Receiver) echo(s string, id int) {
	a, b := rx.c.Get()
	fmt.Printf("%16s(%8d): gonum = %d, value/limit = %d/%d; within/reached/exceeded = %t/%t/%t\n",
		s, id, runtime.NumGoroutine(), a, b, a < b, a == b, a > b)
}
