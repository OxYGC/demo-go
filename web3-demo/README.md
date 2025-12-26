帮我在web3-demo目录下的bitcoin目录初始化一个golang项目，并且帮我生成一个BTC的demo，Demo里面包含生成公私钥，签名方法， 获取链上报文，提现构造并且上链接等功能

把main函数里面的公私钥和地址，还有测试节点配置放到.env文件里面获取
优化：节点配置先从.env里面获取，如果没有就从yml里面获取，类似下面这种:
```yaml
wallet_node:
  eth:
    rpc_url: 'https://eth-holesky.g.alchemy.com/v2/afSCtxPWD3NE5vSjJm2GQ'
    rpc_user: ''
    rpc_pass: ''
    data_api_url: 'https://api-holesky.etherscan.io/api?'
    data_api_key: 'xxxx'
    data_api_token: ''
    time_out: 15
```
