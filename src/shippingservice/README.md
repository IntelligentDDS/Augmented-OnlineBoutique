# Shipping Service

The Shipping service provides price quote, tracking IDs, and the impression of order fulfillment & shipping processes.

failpoint-ctl enable

docker build -t registry.cn-shenzhen.aliyuncs.com/trainticket/hipster-shippingservice:fault .

docker push registry.cn-shenzhen.aliyuncs.com/trainticket/hipster-shippingservice:fault


## Dynamic Inject fault
```
./goFaultInjector -f ShippingGetQuoteLatency -u http://127.0.0.1:12346 -d 60 -p "return(1000)" -c /src/fault.yaml
./goFaultInjector -f ShippingShipOrderMemory -u http://127.0.0.1:12346 -d 60 -c /src/fault.yaml
./goFaultInjector -f ShippingCreateQuoteFromFloatWrite -u http://127.0.0.1:12346 -d 30 -c /src/fault.yaml
./goFaultInjector -f ShippingCreateQuoteFromCountRead -u http://127.0.0.1:12346 -d 30 -c /src/fault.yaml
./goFaultInjector -f ShippingQuoteByCountFloatCPU -u http://127.0.0.1:12346 -d 60 -c /src/fault.yaml
```


service: shipping
faults: 
  - fault: ShippingGetQuoteLatency
    type: latency
    location: main/ShippingGetQuoteLatency
  - fault: ShippingShipOrderMemory
    type: memory
    location: main/ShippingShipOrderMemory
  - fault: ShippingCreateQuoteFromCountRead
    type: read
    location: main/ShippingCreateQuoteFromCountRead
  - fault: ShippingCreateQuoteFromFloatWrite
    type: write
    location: main/ShippingCreateQuoteFromFloatWrite
  - fault: ShippingQuoteByCountFloatCPU
    type: cpu
    location: main/ShippingQuoteByCountFloatCPU
