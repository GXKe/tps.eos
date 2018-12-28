package main

import (
	"fmt"
	"github.com/GXK666/tps.eos/config"
	s "github.com/GXK666/tps.eos/send"
	"golang.org/x/net/context"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	ctxKeyWorkID = 0
)

type job struct {

}

func main()  {

	jobList := make(chan job, 1000)
	ctx, cancel := context.WithCancel(context.Background())

	for i := uint32(0); i < config.Config.Routine; i++ {
		ctx := context.WithValue(ctx, ctxKeyWorkID, i)
		go work(ctx, jobList)
	}

	go func() {
		ticker := time.NewTicker(time.Second / 2)

		for {
			time := <-ticker.C
			fmt.Println(time.String())

			for i := uint32(0); i < config.Config.Tps/2; i++ {
				jobList <- job{}
			}
		}
	}()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan
	cancel()
	time.Sleep(2 * time.Second)
}

func work(ctx context.Context, list chan job)  {
	fmt.Printf("%d \twork run.\n", ctx.Value(ctxKeyWorkID))
	for {
		select {
		case <-ctx.Done():
			fmt.Printf("%d \twork exit\n", ctx.Value(ctxKeyWorkID))
			return
		case <-list:
			err := send(ctx)
			if nil != err {
				fmt.Printf("%d \twork, send err : %v\n", ctx.Value(ctxKeyWorkID), err)
			}
		}
	}
}

func send(ctx context.Context) error {
	//fmt.Printf("%d \twork send http.\n", ctx.Value(ctxKeyWorkID))

	return s.SendHello()
}
