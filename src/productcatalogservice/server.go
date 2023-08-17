// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	//"bytes"
	"context"
	"flag"
	"fmt"

	//"io/ioutil"
	"net"
	"os"

	//"os/signal"
	"strings"
	"sync"

	//"syscall"
	"strconv"
	"time"

	pb "github.com/GoogleCloudPlatform/microservices-demo/src/productcatalogservice/genproto"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	_ "github.com/go-sql-driver/mysql"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"

	// "github.com/uptrace/opentelemetry-go-extra/otelsql"
	// "cloud.google.com/go/profiler"
	//"github.com/golang/protobuf/jsonpb"
	"database/sql"

	"github.com/XSAM/otelsql"
	"github.com/pingcap/failpoint"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	cat          pb.ListProductsResponse
	catalogMutex *sync.Mutex
	log          *logrus.Logger
	extraLatency time.Duration

	port = "3550"

	reloadCatalog bool
	db            *sql.DB
	// Add attribute of pod name and node name
	tracer   = otel.Tracer("product-tracer")
	podName  = os.Getenv("POD_NAME")
	nodeName = os.Getenv("NODE_NAME")
)

// Initializes an OTLP exporter, and configures the corresponding trace and providers.
func initProvider() func() {
	endpoint := os.Getenv("COLLECTOR_ADDR")
	serviceName := os.Getenv("SERVICE_NAME")

	ctx := context.Background()

	res, err := resource.New(ctx,
		resource.WithAttributes(
			// the service name used to display traces in backends
			semconv.ServiceNameKey.String(serviceName),
			attribute.String("PodName", podName),
			attribute.String("NodeName", nodeName),
		),
	)

	if err != nil {
		log.Fatalf("Failed to create resource: %v", err)

	}

	// Set up a trace exporter
	traceExporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithDialOption(grpc.WithBlock()),
	)

	if err != nil {
		log.Fatalf("Failed to create trace exporter: %v", err)
	}

	// Register the trace exporter with a TracerProvider, using a batch
	// span processor to aggregate spans before export.
	bsp := sdktrace.NewBatchSpanProcessor(traceExporter)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)
	otel.SetTracerProvider(tracerProvider)

	// set global propagator to tracecontext (the default is no-op).
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return func() {
		// Shutdown will flush any remaining spans and shut down the exporter.
		if tracerProvider.Shutdown(ctx) != nil {
			log.Fatalf("Failed to shutdown TracerProvider: %v", tracerProvider.Shutdown(ctx))
		}
	}
}

func main() {
	log = logrus.New()
	log.Formatter = &logrus.JSONFormatter{
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "severity",
			logrus.FieldKeyMsg:   "message",
		},
		TimestampFormat: time.RFC3339Nano,
	}
	log.Out = os.Stdout

	shutdown := initProvider()
	defer shutdown()

	catalogMutex = &sync.Mutex{}
	err := initDB()
	if err != nil {
		log.Warnf("could not connect database: %v", err)
	}

	tracer = otel.Tracer("product-tracer")

	flag.Parse()

	reloadCatalog = true

	if os.Getenv("PORT") != "" {
		port = os.Getenv("PORT")
	}
	log.Infof("starting grpc server at :%s", port)
	run(port)
	select {}
}

func run(port string) string {
	l, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatal(err)
	}
	var srv *grpc.Server

	srv = grpc.NewServer(
		grpc.UnaryInterceptor(otelgrpc.UnaryServerInterceptor()),
		grpc.StreamInterceptor(otelgrpc.StreamServerInterceptor()),
	)

	svc := &productCatalog{}

	pb.RegisterProductCatalogServiceServer(srv, svc)
	healthpb.RegisterHealthServer(srv, svc)
	go srv.Serve(l)
	return l.Addr().String()
}

type productCatalog struct{}

func initDB() (err error) {
	mysqlAddr := os.Getenv("MYSQL_ADDR")
	connMaxLifeTime, _ := strconv.Atoi(os.Getenv("ConnMaxLifeTime"))
	maxIdleConns, _ := strconv.Atoi(os.Getenv("mySQLmaxIdleConns"))

	driverName, err := otelsql.Register("mysql", semconv.DBSystemMySQL.Value.AsString())
	if err != nil {
		log.Fatalf("failed to register mysql database: %v", err)
	}

	// connect to mysql
	user := os.Getenv("SQL_USER")
	pwd := os.Getenv("SQL_PASSWORD")
	path := strings.Join([]string{user, ":", pwd, "@tcp(", mysqlAddr, ")/Productdb"}, "")
	db, err = sql.Open(driverName, path)
	if err = db.Ping(); err != nil {
		log.Fatalf("failed to open mysql database: %v", err)
		return err
	}
	db.SetConnMaxLifetime(time.Duration(connMaxLifeTime))
	db.SetMaxIdleConns(maxIdleConns)

	if err = db.Ping(); err != nil {
		log.Fatalf("failed to ping mysql database: %v", err)
		return err
	}
	return nil
}

func readCatalogFile(ctx context.Context, catalog *pb.ListProductsResponse) error {
	ctx, readCatalogSpan := tracer.Start(ctx, "hipstershop.ProductCatalogService/ReadCatalog")
	defer readCatalogSpan.End()

	catalogMutex.Lock()
	defer catalogMutex.Unlock()

	traceId := readCatalogSpan.SpanContext().TraceID()
	spanId := readCatalogSpan.SpanContext().SpanID()

	log.Infof("TraceID: %v SpanID: %v Start query product catalog", traceId, spanId)
	var productList []*pb.Product
	querySQL := "SELECT a.product_id, a.item_name, a.description, a.picture_path, a.categorise_list, b.currencyCode, b.units, b.nanos FROM products a INNER JOIN price b ON a.product_id = b.product_id"
	rows, err := db.QueryContext(ctx, querySQL)

	if err != nil {
		log.Warnf("TraceID: %v SpanID: %v failed to query mysql: %v", traceId, spanId, err)
		return err
	}
	// Parse query results to Product struct
	for rows.Next() {
		var product pb.Product
		var money pb.Money
		var categoriesStr string
		err := rows.Scan(&product.Id, &product.Name, &product.Description, &product.Picture, &categoriesStr, &money.CurrencyCode, &money.Units, &money.Nanos)
		if err != nil {
			log.Warnf("TraceID: %v SpanID: %v failed to scan mysql query result: %v", traceId, spanId, err)
			return err
		}
		product.Categories = strings.Split(categoriesStr, ";")
		product.PriceUsd = &money
		productList = append(productList, &product)
		log.Infof("TraceID: %v SpanID: %v get results: %v", traceId, spanId, product)
	}
	defer rows.Close()
	catalog.Products = productList

	return nil
}

func parseCatalog(ctx context.Context) []*pb.Product {
	ctx, parseCatalogSpan := tracer.Start(ctx, "hipstershop.ProductCatalogService/ParseCatalog")
	defer parseCatalogSpan.End()

	traceId := parseCatalogSpan.SpanContext().TraceID()
	spanId := parseCatalogSpan.SpanContext().SpanID()

	log.Infof("TraceID: %v SpanID: %v Start parse catalog", traceId, spanId)

	if reloadCatalog || len(cat.Products) == 0 {
		err := readCatalogFile(ctx, &cat)

		if err != nil {
			log.Errorf("TraceID: %v SpanID: %v Parse catalog failed", traceId, spanId)
			return []*pb.Product{}
		}
	}

	log.Infof("TraceID: %v SpanID: %v Parse catalog successfully", traceId, spanId)
	return cat.Products
}

func (p *productCatalog) Check(ctx context.Context, req *healthpb.HealthCheckRequest) (*healthpb.HealthCheckResponse, error) {
	return &healthpb.HealthCheckResponse{Status: healthpb.HealthCheckResponse_SERVING}, nil
}

func (p *productCatalog) Watch(req *healthpb.HealthCheckRequest, ws healthpb.Health_WatchServer) error {
	return status.Errorf(codes.Unimplemented, "health check via Watch not implemented")
}

func (p *productCatalog) ListProducts(ctx context.Context, req *pb.Empty) (*pb.ListProductsResponse, error) {
	listProductSpan := trace.SpanFromContext(ctx)
	traceId := listProductSpan.SpanContext().TraceID()
	spanId := listProductSpan.SpanContext().SpanID()

	log.Infof("TraceID: %v SpanID: %v Start list products", traceId, spanId)

	/*
	  Inject cpu consume fault, trigger by ProductListProductsCPU flag
	*/
	if _, _err_ := failpoint.Eval(_curpkg_("ProductListProductsCPU")); _err_ == nil {
		start := time.Now()
		for {
			// break for after duration
			if time.Now().Sub(start).Milliseconds() > 800 {
				break
			}
		}
	}

	listResult := &pb.ListProductsResponse{Products: parseCatalog(ctx)}
	log.Infof("TraceID: %v SpanID: %v List products end", traceId, spanId)

	return listResult, nil
}

func (p *productCatalog) GetProduct(ctx context.Context, req *pb.GetProductRequest) (*pb.Product, error) {
	getProductSpan := trace.SpanFromContext(ctx)
	traceId := getProductSpan.SpanContext().TraceID()
	spanId := getProductSpan.SpanContext().SpanID()

	var found pb.Product
	var money pb.Money
	var categoriesStr string

	log.Infof("TraceID: %v SpanID: %v Query product with name and description %v", traceId, spanId, req.Id)

	querySQL := "SELECT a.product_id, a.item_name, a.description, a.picture_path, a.categorise_list, b.currencyCode, b.units, b.nanos FROM products a INNER JOIN price b ON a.product_id = b.product_id WHERE a.product_id='" + req.Id + "'"
	rows, err := db.QueryContext(ctx, querySQL)

	if err != nil {
		log.Warnf("TraceID: %v SpanID: %v failed to query product by id: %v", traceId, spanId, err)
		return nil, err
	}
	for rows.Next() {
		err := rows.Scan(&found.Id, &found.Name, &found.Description, &found.Picture, &categoriesStr, &money.CurrencyCode, &money.Units, &money.Nanos)
		/*
			Inject Exception Fault, trigger by ProductGetProductException flag
		*/
		if _, _err_ := failpoint.Eval(_curpkg_("ProductGetProductException")); _err_ == nil {
			err = fmt.Errorf("Query product error.")
		}
		if err != nil {
			log.Warnf("TraceID: %v SpanID: %v failed to scan mysql query result: %v", traceId, spanId, err)
			return nil, err
		}
		found.Categories = strings.Split(categoriesStr, ";")
		found.PriceUsd = &money
	}

	defer rows.Close()

	/*
		Inject modify return vaule Fault, trigger by ProductGetProductReturn flag
	*/
	if _, _err_ := failpoint.Eval(_curpkg_("ProductGetProductReturn")); _err_ == nil {
		err = fmt.Errorf("Inject ProductParseCatalogReturn Error")
		log.Errorf("TraceID: %v SpanID: %v No product with ID %v", traceId, spanId, req.Id)
		return nil, status.Errorf(codes.NotFound, "no product with ID %s", req.Id)
	}

	if &found == nil {
		log.Errorf("TraceID: %v SpanID: %v No product with ID %v", traceId, spanId, req.Id)
		return nil, status.Errorf(codes.NotFound, "no product with ID %s", req.Id)
	}

	log.Infof("TraceID: %v SpanID: %v Query product successfully", traceId, spanId)

	return &found, nil
}

func (p *productCatalog) SearchProducts(ctx context.Context, req *pb.SearchProductsRequest) (*pb.SearchProductsResponse, error) {
	searchProductSpan := trace.SpanFromContext(ctx)
	traceId := searchProductSpan.SpanContext().TraceID()
	spanId := searchProductSpan.SpanContext().SpanID()

	log.Infof("TraceID: %v SpanID: %v Search product with name and description %v", traceId, spanId, strings.ToLower(req.Query))

	// Intepret query as a substring match in name or description.
	var ps []*pb.Product
	for _, p := range parseCatalog(ctx) {
		if strings.Contains(strings.ToLower(p.Name), strings.ToLower(req.Query)) ||
			strings.Contains(strings.ToLower(p.Description), strings.ToLower(req.Query)) {
			ps = append(ps, p)
			log.Infof("TraceID: %v SpanID: %v Find product %v", traceId, spanId, p)
		}
	}

	if len(ps) == 0 {
		log.Warnf("TraceID: %v SpanID: %v No product with %v", traceId, spanId, strings.ToLower(req.Query))
	}

	return &pb.SearchProductsResponse{Results: ps}, nil
}
