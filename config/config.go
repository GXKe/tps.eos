package config

import (
	"encoding/json"
	"io/ioutil"
)

var(
	Config Config_t
)

func init()  {
	data, err := ioutil.ReadFile("./config.json")
	if nil != err {
		panic(err)
	}

	err = json.Unmarshal(data, &Config)
	if nil != err {
		panic(err)
	}
}


type Config_t struct {
	PrivKey string `json:"PrivKey"`
	AccountName string `json:"AccountName"`
	NodeList []string `json:"NodeList"` 
	Tps  uint32 `json:"Tps"`
	Routine uint32 `json:"Routine"`
}