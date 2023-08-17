# checkoutservice

failpoint-ctl enable

docker build -t registry.cn-shenzhen.aliyuncs.com/trainticket/hipster-checkoutservice:fault .

docker push registry.cn-shenzhen.aliyuncs.com/trainticket/hipster-checkoutservice:fault


## Dynamic Inject fault
```
./goFaultInjector -f CheckoutGetUserCartLatency -u http://127.0.0.1:12346 -d 60 -p "return(1000)" -c /src/fault.yaml
./goFaultInjector -f CheckoutquoteShippingLatency -u http://127.0.0.1:12346 -d 60 -p "return(1000)" -c /src/fault.yaml
./goFaultInjector -f CheckoutPrepOrderItemsMemory -u http://127.0.0.1:12346 -d 60 -c /src/fault.yaml
./goFaultInjector -f CheckoutSendOrderConfirmationReturn -u http://127.0.0.1:12346 -d 60 -c /src/fault.yaml
./goFaultInjector -f CheckoutSendOrderConfirmationException -u http://127.0.0.1:12346 -d 30 -c /src/fault.yaml
./goFaultInjector -f CheckoutPrepOrderItemsWrite -u http://127.0.0.1:12346 -d 30 -c /src/fault.yaml
./goFaultInjector -f CheckoutPrepOrderItemsRead -u http://127.0.0.1:12346 -d 30 -c /src/fault.yaml
./goFaultInjector -f CheckoutPrepOrderItemsCPU -u http://127.0.0.1:12346 -d 60 -c /src/fault.yaml
```
