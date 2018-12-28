package loger

import (
	"github.com/GXK666/tps.eos/config"
	"os"
	"time"
	"log"
)

var txidLoger *log.Logger

func init()  {
	if !config.Config.OutputTxid {
		return
	}
	file := "./txids_" + time.Now().Format("2006-01-02_150405") + ".txt"
	logFile, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0766)
	if nil != err {
		panic(err)
	}

	txidLoger = log.New(logFile, "",0)
	if nil == txidLoger {
		panic("TxidLoger init fail")
	}
}

func LogTxid(txid string) {
	if nil != txidLoger {
		txidLoger.Println(txid)
	}
}




