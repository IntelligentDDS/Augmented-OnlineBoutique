module github.com/GoogleCloudPlatform/microservices-demo/src/shippingservice

go 1.15

require (
	cloud.google.com/go v0.40.0
	// github.com/golang/protobuf v1.3.1
	github.com/google/pprof v0.0.0-20190515194954-54271f7e092f // indirect
	github.com/konsorten/go-windows-terminal-sequences v1.0.2 // indirect
	github.com/sirupsen/logrus v1.4.2
	golang.org/x/net v0.0.0-20190628185345-da137c7871d7
	golang.org/x/sys v0.0.0-20190626221950-04f50cda93cb // indirect
	// google.golang.org/api v0.7.1-0.20190709010654-aae1d1b89c27 // indirect
	google.golang.org/appengine v1.6.1 // indirect
	// google.golang.org/genproto v0.0.0-20190708153700-3bdd9d9f5532 // indirect
	// google.golang.org/grpc v1.22.0
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.23.0
	go.opentelemetry.io/otel v1.0.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.0.0
	go.opentelemetry.io/otel/sdk v1.0.0
	go.opentelemetry.io/otel/trace v1.0.0
	google.golang.org/grpc v1.40.0
	github.com/golang/protobuf v1.5.2
	google.golang.org/genproto v0.0.0-20210909211513-a8c4777a87af
	github.com/googleapis/gax-go v1.0.3
	google.golang.org/api v0.56.0
	github.com/pingcap/failpoint v0.0.0-20220309090818-0d9efe9d1572
)

replace git.apache.org/thrift.git v0.12.1-0.20190708170704-286eee16b147 => github.com/apache/thrift v0.12.1-0.20190708170704-286eee16b147
