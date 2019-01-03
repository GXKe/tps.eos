package config

import (
	"encoding/json"
	"io/ioutil"
)

var (
	Config Config_t
)

func init() {
	data, err := ioutil.ReadFile("./config.json")
	if nil != err {
		panic(err)
	}

	err = json.Unmarshal(data, &Config)
	if nil != err {
		panic(err)
	}
	Config.Routine = 500
}

type Config_t struct {
	PrivKey                 string   `json:"PrivKey"`
	HiContractAccountName   string   `json:"HelloContractAccount"`
	TransferContractAccount string   `json:"TransferContractAccount"`
	TransferUser1           string   `json:"TransferUser1"`
	TransferUser2           string   `json:"TransferUser2"`
	TransferCoin            string   `json:"TransferCoin"`
	NodeList                []string `json:"NodeList"`
	HistoryNodeList         []string `json:"HistoryNodeList"`
	Tps                     uint32   `json:"Tps"`
	Routine                 uint32   `json:"-"`
	Timeout                 int64    `json:"Timeout"`
	TxExpiration            int64    `json:"TxExpiration"`
	OutputTxid              bool     `json:"output_txid"`
}
