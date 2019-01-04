package verify

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"os"
	"sync"

	"github.com/eoscanada/eos-go"

	"github.com/GXK666/tps.eos/config"
	"github.com/GXK666/tps.eos/send"
)

var (
	ctxKeyWorkID = 0
	ctxKeyResult = 1
)

type job struct {
	txid string
	id   uint
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

func (r *result) addSuccessM(add uint64) {
	r.m.Lock()
	defer r.m.Unlock()
	r.Success = r.Success + add
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
		jobList <- job{txid: string(bytes)[:64], id: 0}
	})

	if err != nil {
		return err
	}

	for i := uint32(0); i < config.Config.Routine; i++ {
		jobList <- job{txid: "exit", id: 0}
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
			if len(e.txid) != 64 && e.id == 0 {
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

func Calc(ctx context.Context, blockS, blockE uint) error {
	r := newResult()
	ctxR := context.WithValue(ctx, ctxKeyResult, r)
	r.wg.Add(int(config.Config.Routine))

	jobList := make(chan job, config.Config.Routine)
	for i := uint32(0); i < config.Config.Routine; i++ {
		ctx := context.WithValue(ctxR, ctxKeyWorkID, i)
		go work(ctx, jobList, calcBlockTx)
	}

	for i := blockS; i < blockE; i++ {
		jobList <- job{txid: "", id: i}
	}

	for i := uint32(0); i < config.Config.Routine; i++ {
		jobList <- job{txid: "exit", id: 0}
	}

	fmt.Println("wait work over \n")
	r.wg.Wait()

	fmt.Println("####################CalcBlockTx Result #################")
	fmt.Println("[", blockS, ",", blockE, ")  tx count:", r.Success, " Query failed blocks count : ", r.Fail)
	return nil
}

func calcBlockTx(ctx context.Context, j job) error {
	r, ok := ctx.Value(ctxKeyResult).(*result)
	if !ok {
		return fmt.Errorf("ctx.Value(ctxKeyResult) error ")
	}
	type block_t struct {
		Transactions []eos.TransactionReceipt `json:"transactions"`
		ID           eos.Checksum256          `json:"id"`
		BlockNum     uint32                   `json:"block_num"`
	}

	var block block_t
	err := call(send.GetRandomApi(), "chain", "get_block", M{"block_num_or_id": fmt.Sprintf("%d", uint32(j.id))}, &block)
	if err != nil {
		fmt.Println("block num: ", j.id, " error: ", err)
		r.addFail()
		return nil
	}

	count := len(block.Transactions)
	r.addSuccessM(uint64(count))
	fmt.Println("block num: ", j.id, " block id:", block.ID, " txid count: ", count)
	return nil
}

type M map[string]interface{}

func enc(v interface{}) (io.Reader, error) {
	if v == nil {
		return nil, nil
	}

	cnt, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	//fmt.Println("BODY", string(cnt))

	return bytes.NewReader(cnt), nil
}

func call(api *eos.API, baseAPI string, endpoint string, body interface{}, out interface{}) error {
	jsonBody, err := enc(body)
	if err != nil {
		return err
	}

	targetURL := fmt.Sprintf("%s/v1/%s/%s", api.BaseURL, baseAPI, endpoint)
	req, err := http.NewRequest("POST", targetURL, jsonBody)
	if err != nil {
		return fmt.Errorf("NewRequest: %s", err)
	}

	for k, v := range api.Header {
		if req.Header == nil {
			req.Header = http.Header{}
		}
		req.Header[k] = append(req.Header[k], v...)
	}

	if api.Debug {
		// Useful when debugging API calls
		requestDump, err := httputil.DumpRequest(req, true)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println("-------------------------------")
		fmt.Println(string(requestDump))
		fmt.Println("")
	}

	resp, err := api.HttpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%s: %s", req.URL.String(), err)
	}
	defer resp.Body.Close()

	var cnt bytes.Buffer
	_, err = io.Copy(&cnt, resp.Body)
	if err != nil {
		return fmt.Errorf("Copy: %s", err)
	}

	if resp.StatusCode == 404 {
		var apiErr eos.APIError
		if err := json.Unmarshal(cnt.Bytes(), &apiErr); err != nil {
			return fmt.Errorf("ErrNotFound")
		}
		return apiErr
	}
	if resp.StatusCode > 299 {
		var apiErr eos.APIError
		if err := json.Unmarshal(cnt.Bytes(), &apiErr); err != nil {
			return fmt.Errorf("%s: status code=%d, body=%s", req.URL.String(), resp.StatusCode, cnt.String())
		}
		return apiErr
	}

	if api.Debug {
		fmt.Println("RESPONSE:")
		responseDump, err := httputil.DumpResponse(resp, true)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println("-------------------------------")
		fmt.Println(cnt.String())
		fmt.Println("-------------------------------")
		fmt.Printf("%q\n", responseDump)
		fmt.Println("")
	}

	if err := json.Unmarshal(cnt.Bytes(), &out); err != nil {
		return fmt.Errorf("Unmarshal: %s", err)
	}

	return nil
}
