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
)

var (
	nodeApi []*eos.API
)

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
			nodeApi = append(nodeApi, api)
		}
	}

	if len(nodeApi) ==0 {
		panic("node Api[] is null")
	}
	fmt.Println("init node api success")
}



type Hello struct {
	User eos.AccountName `json "user"`
}

var api = eos.New(config.Config.NodeList[0])

func NewHelloAction() *eos.Action {
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

func SendHello() error {

	hi := NewHelloAction()

	if len(nodeApi) == 0 {
		fmt.Print("node Api list is nil")
		return fmt.Errorf("node Api list is nil ")
	}
	nodeIdx := uint32(0)
	if len(nodeApi) > 1 {
		nodeIdx = rand.Uint32() % uint32(len(nodeApi))
	}

	nonce, _:=	uuid.NewV4()
	rsp, err := nodeApi[nodeIdx].SignPushActions(hi, system.NewNonce(nonce.String()))
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

