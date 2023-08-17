# frontservice

failpoint-ctl enable

docker build -t registry.cn-shenzhen.aliyuncs.com/trainticket/hipster-frontend:fault .

docker push registry.cn-shenzhen.aliyuncs.com/trainticket/hipster-frontend:fault


## Dynamic Inject fault
```
./goFaultInjector -f FrontSetCurrencyHandlerLatency -u http://127.0.0.1:12346 -d 60 -p "return(1000)" -c /src/fault.yaml
./goFaultInjector -f FrontSetCurrencyHandlerMemory -u http://127.0.0.1:12346 -d 60 -c /src/fault.yaml
./goFaultInjector -f FrontProductHandlerReturn -u http://127.0.0.1:12346 -d 60 -c /src/fault.yaml
./goFaultInjector -f FrontPlaceOrderHandlerException -u http://127.0.0.1:12346 -d 30 -c /src/fault.yaml
./goFaultInjector -f FrontChooseAdWrite -u http://127.0.0.1:12346 -d 30 -c /src/fault.yaml
./goFaultInjector -f FrontSetCurrencyHandlerRead -u http://127.0.0.1:12346 -d 30 -c /src/fault.yaml
./goFaultInjector -f FrontChooseAdCPU -u http://127.0.0.1:12346 -d 60 -c /src/fault.yaml
```