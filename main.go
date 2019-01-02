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
	sendType := ""
	flag.StringVar(&file, "f", "", "txid file name")
	flag.StringVar(&sendType, "s", "hi", "send transfer type: hi,transfer")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	if len(file) > 0  {
		verify.VerifyTxid(ctx,file)
		return
	} else {
		switch sendType {
		case "hi":
			send.Run(ctx, send.SEND_HI)
		case "transfer":
			send.Run(ctx, send.SEND_TRANSFER)
		default:
			panic("send Type error!")
		}
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan
	cancel()
	time.Sleep(3 * time.Second)
}
