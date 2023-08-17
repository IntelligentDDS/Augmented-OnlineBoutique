# productcatalogservice

failpoint-ctl enable

docker build -t registry.cn-shenzhen.aliyuncs.com/trainticket/hipster-productcatalogservice:fault .

docker push registry.cn-shenzhen.aliyuncs.com/trainticket/hipster-productcatalogservice:fault


## Dynamic Inject fault
```
./goFaultInjector -f ProductReadCatalogLatency -u http://127.0.0.1:12346 -d 60 -p "return(1000)" -c /src/fault.yaml
./goFaultInjector -f ProductReadCatalogMemory -u http://127.0.0.1:12346 -d 60 -c /src/fault.yaml
./goFaultInjector -f ProductParseCatalogReturn -u http://127.0.0.1:12346 -d 60 -c /src/fault.yaml
./goFaultInjector -f ProductGetProductException -u http://127.0.0.1:12346 -d 30 -c /src/fault.yaml
./goFaultInjector -f ProductGetProductLatency -u http://127.0.0.1:12346 -d 30 -p "return(1000)" -c /src/fault.yaml
./goFaultInjector -f ProductListProductsWrite -u http://127.0.0.1:12346 -d 30 -c /src/fault.yaml
./goFaultInjector -f ProductListProductsRead -u http://127.0.0.1:12346 -d 30 -c /src/fault.yaml
./goFaultInjector -f ProductListProductsLatency -u http://127.0.0.1:12346 -d 30 -p "return(1000)" -c /src/fault.yaml
./goFaultInjector -f ProductListProductsCPU -u http://127.0.0.1:12346 -d 60 -c /src/fault.yaml
```
