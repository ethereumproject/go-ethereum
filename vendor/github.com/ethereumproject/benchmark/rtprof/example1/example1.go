package main

import (
	"github.com/eth-classic/benchmark/rtprof"
	"os"
	"os/signal"
	"time"
)

func main() {
	rtppf.Start(5*time.Second, 8082)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	signal.Stop(c)
	rtppf.Stop()
}
