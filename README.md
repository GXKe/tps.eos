# tps.eos
tps test for eos


Need to deploy hello contract, and have enough cpu, net resources

固定读取同级目录下config.json配置

./tps.eos 启动TPS输出，默认输出hi交易

./tps.eos -s transfer 输出transfer交易

./top.eos -f txids.file  校验txid是否存在于区块链上，依赖HistoryNode配置