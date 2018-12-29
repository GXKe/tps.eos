package send

import (
	"fmt"
	"github.com/GXK666/tps.eos/config"
	"github.com/GXK666/tps.eos/loger"
	"github.com/eoscanada/eos-go"
	"github.com/eoscanada/eos-go/ecc"
	"github.com/eoscanada/eos-go/system"
	"github.com/satori/go.uuid"
	"math/rand"
	"time"
	"context"
)

var (
	nodeApiList []*eos.API
	historyNodeApiList []*eos.API
)
var (
	ctxKeyWorkID = 0
)

type job struct {

}
type Hello struct {
	User eos.AccountName `json "user"`
}

func init(){

	privKey, err := ecc.NewPrivateKey(config.Config.PrivKey)
	if nil != err {
		panic("config privkey err")
	}
	pubKey := privKey.PublicKey()

	keyBag := eos.NewKeyBag()
	if err := keyBag.Add(config.Config.PrivKey); err != nil {
		fmt.Print("Couldn't load private key:", err)
	}


	for _, n := range config.Config.NodeList{
		fmt.Println("start link node api: ", n)
		api := eos.New(n)
		if _, err := api.GetInfo() ; err != nil {
			fmt.Println("init node api error : ", n, " ", err)
		} else {
			api.Signer = keyBag
			api.SetCustomGetRequiredKeys(func(tx *eos.Transaction) (keys []ecc.PublicKey, e error) {
				return []ecc.PublicKey{pubKey}, nil
			})
			nodeApiList = append(nodeApiList, api)
		}
	}

	for _, n := range config.Config.HistoryNodeList{
		fmt.Println("start link node api: ", n)
		api := eos.New(n)
		if _, err := api.GetInfo() ; err != nil {
			fmt.Println("init node api error : ", n, " ", err)
		} else {
			api.Signer = keyBag
			api.SetCustomGetRequiredKeys(func(tx *eos.Transaction) (keys []ecc.PublicKey, e error) {
				return []ecc.PublicKey{pubKey}, nil
			})
			nodeApiList = append(nodeApiList, api)
			historyNodeApiList = append(historyNodeApiList, api)
		}
	}

	if len(nodeApiList) ==0 {
		panic("node Api[] is null")
	}
	fmt.Println("init node api success")
}



func Run(ctx context.Context)  {
	jobList := make(chan job, 1000)
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

}


func newHelloAction() *eos.Action {
	return &eos.Action{
		Account: eos.AN(config.Config.AccountName),
		Name:    eos.ActN("hi"),
		Authorization: []eos.PermissionLevel{
			{Actor: eos.AN(config.Config.AccountName), Permission: eos.PN("active")},
		},
		ActionData: eos.NewActionData(Hello{
			User:eos.AN(config.Config.AccountName),
		}),
	}
}

func GetRandomApi() (*eos.API) {
	if len(nodeApiList) == 0 {
		panic("node Api list is nil ")
	}
	nodeIdx := uint32(0)
	if len(nodeApiList) > 1 {
		nodeIdx = rand.Uint32() % uint32(len(nodeApiList))
	}
	return nodeApiList[nodeIdx]
}

func GetRandomHistoryApi() (*eos.API) {
	if len(historyNodeApiList) == 0 {
		panic("history node Api list is nil ")
	}
	nodeIdx := uint32(0)
	if len(historyNodeApiList) > 1 {
		nodeIdx = rand.Uint32() % uint32(len(historyNodeApiList ))
	}
	return historyNodeApiList[nodeIdx]
}

func sendHello() error {

	nonce, _:=	uuid.NewV4()
	rsp, err := GetRandomApi().SignPushActions(newHelloAction(), system.NewNonce(nonce.String()))
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


func work(ctx context.Context, list chan job)  {
	fmt.Printf("%d \twork run.\n", ctx.Value(ctxKeyWorkID))
	for {
		select {
		case <-ctx.Done():
			fmt.Printf("%d \twork exit\n", ctx.Value(ctxKeyWorkID))
			return
		case <-list:
			err := sendHello()
			if nil != err {
				fmt.Printf("%d \twork, send err : %v\n", ctx.Value(ctxKeyWorkID), err)
			}
		}
	}
}
