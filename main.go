package main

import (
	"flag"
	"github.com/GXK666/tps.eos/send"
	"github.com/GXK666/tps.eos/verify"
	"golang.org/x/net/context"
	"os"
	"os/signal"
	"syscall"
	"time"
)


func main()  {
	file := ""
	flag.StringVar(&file, "f", "", "txid file name")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	if len(file) > 0  {
		verify.VerifyTxid(ctx,file)
		return
	} else {
		send.Run(ctx)
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan
	cancel()
	time.Sleep(3 * time.Second)
}
