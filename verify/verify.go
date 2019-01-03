package verify

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/GXK666/tps.eos/config"
	"github.com/GXK666/tps.eos/send"
)

var (
	ctxKeyWorkID = 0
	ctxKeyResult = 1
)

type job struct {
	txid string
}

type result struct {
	Total   uint64
	Fail    uint64
	Success uint64

	m  *sync.Mutex
	wg *sync.WaitGroup
}

func newResult() *result {
	return &result{m: new(sync.Mutex), wg: new(sync.WaitGroup)}
}

func (r *result) addTotal() {
	//r.m.Lock()
	//defer r.m.Unlock()
	r.Total = r.Total + 1
}

func (r *result) addFail() {
	r.m.Lock()
	defer r.m.Unlock()
	r.Fail = r.Fail + 1
}

func (r *result) addSuccess() {
	r.m.Lock()
	defer r.m.Unlock()
	r.Success = r.Success + 1
}

func (r *result) Print() {
	fmt.Printf("Total %d, Success %d, Fail %d, Wait %d,  success %f% \n",
		r.Total, r.Success, r.Fail, r.Total-r.Success-r.Fail, float32(100*r.Success)/float32(r.Total))
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func VerifyTxid(ctx context.Context, file string) error {
	r := newResult()
	ctxR := context.WithValue(ctx, ctxKeyResult, r)
	r.wg.Add(int(config.Config.Routine))

	jobList := make(chan job, config.Config.Routine)
	for i := uint32(0); i < config.Config.Routine; i++ {
		ctx := context.WithValue(ctxR, ctxKeyWorkID, i)
		go work(ctx, jobList, sendVerify)
	}

	err := readLine(ctx, file, func(bytes []byte) {
		if len(bytes) == 0 {
			return
		}
		r.addTotal()
		jobList <- job{txid: string(bytes)[:64]}
	})

	if err != nil {
		return err
	}

	for i := uint32(0); i < config.Config.Routine; i++ {
		jobList <- job{txid: "exit"}
	}

	r.Print()
	fmt.Println("wait work over \n")
	r.wg.Wait()

	fmt.Println("####################VerifyTxid Result #################")
	r.Print()
	return nil
}

func readLine(ctx context.Context, filePth string, hookfn func([]byte)) error {
	f, err := os.Open(filePth)
	if nil != err {
		return err
	}
	defer f.Close()

	bfRd := bufio.NewReader(f)
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			line, err := bfRd.ReadBytes('\n')
			hookfn(line)
			if err != nil {
				if err == io.EOF {
					return nil
				}
				return err
			}
		}
	}
	return nil
}

func work(ctx context.Context, list chan job, hook func(context.Context, job) error) {
	fmt.Printf("%d \twork run.\n", ctx.Value(ctxKeyWorkID))
	r, ok := ctx.Value(ctxKeyResult).(*result)
	if !ok {
		panic("ctx.Value(ctxKeyResult) error ")
	}
	defer r.wg.Done()

	for {
		select {
		case <-ctx.Done():
			fmt.Printf("%d \twork exit\n", ctx.Value(ctxKeyWorkID))
			return
		case e, ok := <-list:
			if !ok {
				fmt.Println(ctx.Value(ctxKeyWorkID), "<-job chan  fail")
			}
			if len(e.txid) != 64 {
				fmt.Printf("%d \twork exit\n", ctx.Value(ctxKeyWorkID))
				return // work exit
			}
			err := hook(ctx, e)
			if nil != err {
				fmt.Printf("%d \twork error: %v\n", ctx.Value(ctxKeyWorkID), err)
			}
		}
	}
}

func sendVerify(ctx context.Context, j job) error {
	r, ok := ctx.Value(ctxKeyResult).(*result)
	if !ok {
		return fmt.Errorf("ctx.Value(ctxKeyResult) error ")
	}
	_, err := send.GetRandomHistoryApi().GetTransaction(j.txid)
	if err != nil {
		fmt.Println("sendVerify txid: ", j.txid, " error: ", err)
		r.addFail()
		return nil
	}

	r.addSuccess()
	fmt.Println("success txid: ", j.txid)
	return nil
}
