package send

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/GXK666/tps.eos/config"
	"github.com/GXK666/tps.eos/loger"
	"github.com/eoscanada/eos-go"
	"github.com/eoscanada/eos-go/ecc"
	"github.com/eoscanada/eos-go/system"
	"github.com/eoscanada/eos-go/token"
	"github.com/satori/go.uuid"
)

var (
	nodeApiList        []*eos.API
	historyNodeApiList []*eos.API
)
var (
	ctxKeyWorkID = 0
)

type SendType int

const (
	SEND_ERR SendType = iota
	SEND_HI
	SEND_TRANSFER
)

type job struct {
	send SendType
	num  uint32
}
type Hello struct {
	User eos.AccountName `json "user"`
}

func init() {

	privKey, err := ecc.NewPrivateKey(config.Config.PrivKey)
	if nil != err {
		panic("config privkey err")
	}
	pubKey := privKey.PublicKey()

	keyBag := eos.NewKeyBag()
	if err := keyBag.Add(config.Config.PrivKey); err != nil {
		fmt.Print("Couldn't load private key:", err)
	}

	for _, n := range config.Config.NodeList {
		fmt.Println("start link node api: ", n)
		api := eos.New(n)
		if _, err := api.GetInfo(); err != nil {
			fmt.Println("init node api error : ", n, " ", err)
		} else {
			api.Signer = keyBag
			api.SetCustomGetRequiredKeys(func(tx *eos.Transaction) (keys []ecc.PublicKey, e error) {
				return []ecc.PublicKey{pubKey}, nil
			})
			api.HttpClient.Timeout = time.Duration(config.Config.Timeout) * time.Second
			//api.Debug = true
			nodeApiList = append(nodeApiList, api)
		}
	}

	for _, n := range config.Config.HistoryNodeList {
		fmt.Println("start link node api: ", n)
		api := eos.New(n)
		if _, err := api.GetInfo(); err != nil {
			fmt.Println("init node api error : ", n, " ", err)
		} else {
			api.Signer = keyBag
			api.SetCustomGetRequiredKeys(func(tx *eos.Transaction) (keys []ecc.PublicKey, e error) {
				return []ecc.PublicKey{pubKey}, nil
			})
			api.HttpClient.Timeout = time.Duration(config.Config.Timeout) * time.Second
			//api.Debug = true
			nodeApiList = append(nodeApiList, api)
			historyNodeApiList = append(historyNodeApiList, api)
		}
	}

	if len(nodeApiList) == 0 {
		panic("node Api[] is null")
	}
	fmt.Println("init node api success")
}

func getSendType(s string) (SendType, error) {
	switch s {
	case "hi":
		return SEND_HI, nil
	case "transfer":
		return SEND_TRANSFER, nil
	default:
		return SEND_ERR, fmt.Errorf("no support send msg type")
	}
}

func Run(ctx context.Context, s string) error {
	send, err := getSendType(s)
	if nil != err {
		return err
	}
	fmt.Printf("%#v", send)

	jobList := make(chan job, 1000)
	for i := uint32(0); i < config.Config.Routine; i++ {
		ctx := context.WithValue(ctx, ctxKeyWorkID, i)
		go work(ctx, jobList, sendTx)
	}

	go func() {
		ticker := time.NewTicker(time.Second / 2)

		for {
			time := <-ticker.C
			fmt.Println(time.String())

			for i := uint32(0); i < config.Config.Tps/2; i++ {
				jobList <- job{num: i, send: send}
			}
		}
	}()

	return nil
}

func newHelloAction() *eos.Action {
	return &eos.Action{
		Account: eos.AN(config.Config.HiContractAccountName),
		Name:    eos.ActN("hi"),
		Authorization: []eos.PermissionLevel{
			{Actor: eos.AN(config.Config.HiContractAccountName), Permission: eos.PN("active")},
		},
		ActionData: eos.NewActionData(Hello{
			User: eos.AN(config.Config.HiContractAccountName),
		}),
	}
}

func newTransferAction(num uint32) *eos.Action {
	from, to := eos.AN(config.Config.TransferUser1), eos.AN(config.Config.TransferUser2)
	if num%uint32(2) > 0 {
		to, from = from, to
	}

	asset, err := eos.NewAsset(config.Config.TransferCoin)
	if nil != err {
		panic("config.Config.TransferCoin err")
	}

	return &eos.Action{
		Account: eos.AN(config.Config.TransferContractAccount),
		Name:    eos.ActN("transfer"),
		Authorization: []eos.PermissionLevel{
			{Actor: from, Permission: eos.PN("active")},
		},
		ActionData: eos.NewActionData(token.Transfer{
			From:     from,
			To:       to,
			Quantity: asset,
			Memo:     "",
		}),
	}
}

func GetRandomApi() *eos.API {
	if len(nodeApiList) == 0 {
		panic("node Api list is nil ")
	}
	nodeIdx := uint32(0)
	if len(nodeApiList) > 1 {
		nodeIdx = rand.Uint32() % uint32(len(nodeApiList))
	}
	return nodeApiList[nodeIdx]
}

func GetRandomHistoryApi() *eos.API {
	if len(historyNodeApiList) == 0 {
		panic("history node Api list is nil ")
	}
	nodeIdx := uint32(0)
	if len(historyNodeApiList) > 1 {
		nodeIdx = rand.Uint32() % uint32(len(historyNodeApiList))
	}
	return historyNodeApiList[nodeIdx]
}

func newSendAction(job2 job) *eos.Action {
	switch job2.send {
	case SEND_HI:
		return newHelloAction()
	case SEND_TRANSFER:
		return newTransferAction(job2.num)
	}
	return nil
}

func signPushActions(actions []*eos.Action) (out *eos.PushTransactionFullResp, err error) {

	api := GetRandomApi()
	opts := &eos.TxOptions{}
	if err := opts.FillFromChain(api); err != nil {
		return nil, err
	}

	tx := eos.NewTransaction(actions, opts)
	tx.SetExpiration(time.Duration(config.Config.TxExpiration) * time.Second)

	return api.SignPushTransaction(tx, opts.ChainID, opts.Compress)
}

func sendTx(ctx context.Context, job2 job) error {
	nonce, _ := uuid.NewV4()
	actions := []*eos.Action{newSendAction(job2), system.NewNonce(nonce.String())}
	rsp, err := signPushActions(actions)
	if nil != err {
		return fmt.Errorf("rsp error :%v", err)
	}
	if rsp.StatusCode != "" {
		return fmt.Errorf("rsp : %#v", rsp)
	}

	fmt.Println("\t success , send trx id: ", rsp.TransactionID)

	loger.LogTxid(rsp.TransactionID)

	return nil
}

func work(ctx context.Context, list chan job, hook func(context.Context, job) error) {
	fmt.Printf("%d \twork run.\n", ctx.Value(ctxKeyWorkID))
	for {
		select {
		case <-ctx.Done():
			fmt.Printf("%d \twork exit\n", ctx.Value(ctxKeyWorkID))
			return
		case e, ok := <-list:
			if !ok {
				fmt.Println(ctx.Value(ctxKeyWorkID), "<-job chan  fail")
			}
			err := hook(ctx, e)
			if nil != err {
				fmt.Printf("%d \twork error: %v\n", ctx.Value(ctxKeyWorkID), err)
			}
		}
	}
}
