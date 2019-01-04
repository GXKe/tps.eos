package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/GXK666/tps.eos/send"
	"github.com/GXK666/tps.eos/verify"
	"golang.org/x/net/context"
)

func main() {
	file := ""
	sendType := ""
	flag.StringVar(&file, "f", "", "txid file name")
	flag.StringVar(&sendType, "s", "hi", "send transfer type: hi,transfer")
	blockStart := flag.Uint("cs", 0, "start block num for calc tx")
	blockEnd := flag.Uint("ce", 0, "end block num for calc tx")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	if len(file) > 0 {
		verify.VerifyTxid(ctx, file)
		return
	} else {
		if blockEnd != nil && blockStart != nil && *blockStart > 0 && *blockEnd > *blockStart {
			verify.Calc(ctx, *blockStart, *blockEnd)
		} else if err := send.Run(ctx, sendType); nil != err {
			panic(err)
		}

	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan
	cancel()
	time.Sleep(3 * time.Second)
}
