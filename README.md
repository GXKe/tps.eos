# tps.eos
tps test for eos


Need to deploy hello contract, and have enough cpu, net resources

固定读取同级目录下config.json配置，配置中所用到的账号请设置为同一私钥。

./tps.eos 启动TPS输出，默认输出hi交易

./tps.eos -s transfer 输出transfer交易

./top.eos -f txids.file  校验txid是否存在于区块链上，依赖HistoryNode配置


 ./tps.eos -cs 10101 -ce  10401  统计区块内交易数量