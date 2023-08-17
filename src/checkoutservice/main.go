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
	"context"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/pingcap/failpoint"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"

	// "go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/trace"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"

	pb "github.com/GoogleCloudPlatform/microservices-demo/src/checkoutservice/genproto"
	money "github.com/GoogleCloudPlatform/microservices-demo/src/checkoutservice/money"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

const (
	listenPort  = "5050"
	usdCurrency = "USD"
)

var log *logrus.Logger

type checkoutService struct {
	productCatalogSvcAddr string
	cartSvcAddr           string
	currencySvcAddr       string
	shippingSvcAddr       string
	emailSvcAddr          string
	paymentSvcAddr        string
}

var (
	// Add attribute of pod name and node name
	tracer   = otel.Tracer("checkout-tracer")
	podName  = os.Getenv("POD_NAME")
	nodeName = os.Getenv("NODE_NAME")

	// labels represent additional key-value descriptors that can be bound to a
	// metric observer or recorder.
	// commonLabels = []attribute.KeyValue{
	// 	attribute.String("PodName", podName),
	// 	attribute.String("NodeName", nodeName)}
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
		log.Errorf("Failed to create resource: %v", err)
	}

	// Set up a trace exporter
	traceExporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithDialOption(grpc.WithBlock()),
	)

	if err != nil {
		log.Errorf("Failed to create trace exporter: %v", err)
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
			log.Errorf("Failed to shutdown TracerProvider: %v", tracerProvider.Shutdown(ctx))
		}
	}
}

func main() {
	log = logrus.New()
	log.Level = logrus.DebugLevel
	log.Formatter = &logrus.JSONFormatter{
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "severity",
			logrus.FieldKeyMsg:   "message",
		},
		TimestampFormat: time.RFC3339Nano,
	}
	log.Out = os.Stdout

	port := listenPort
	if os.Getenv("PORT") != "" {
		port = os.Getenv("PORT")
	}

	shutdown := initProvider()
	defer shutdown()
	tracer = otel.Tracer("checkout-tracer")

	svc := new(checkoutService)
	mustMapEnv(&svc.shippingSvcAddr, "SHIPPING_SERVICE_ADDR")
	mustMapEnv(&svc.productCatalogSvcAddr, "PRODUCT_CATALOG_SERVICE_ADDR")
	mustMapEnv(&svc.cartSvcAddr, "CART_SERVICE_ADDR")
	mustMapEnv(&svc.currencySvcAddr, "CURRENCY_SERVICE_ADDR")
	mustMapEnv(&svc.emailSvcAddr, "EMAIL_SERVICE_ADDR")
	mustMapEnv(&svc.paymentSvcAddr, "PAYMENT_SERVICE_ADDR")

	log.Infof("service config: %+v", svc)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatal(err)
	}

	var srv *grpc.Server

	srv = grpc.NewServer(
		grpc.UnaryInterceptor(otelgrpc.UnaryServerInterceptor()),
		grpc.StreamInterceptor(otelgrpc.StreamServerInterceptor()),
	)

	pb.RegisterCheckoutServiceServer(srv, svc)
	healthpb.RegisterHealthServer(srv, svc)
	log.Infof("starting to listen on tcp: %q", lis.Addr().String())
	err = srv.Serve(lis)
	log.Fatal(err)
}

func mustMapEnv(target *string, envKey string) {
	v := os.Getenv(envKey)
	if v == "" {
		panic(fmt.Sprintf("environment variable %q not set", envKey))
	}
	*target = v
}

func (cs *checkoutService) Check(ctx context.Context, req *healthpb.HealthCheckRequest) (*healthpb.HealthCheckResponse, error) {
	return &healthpb.HealthCheckResponse{Status: healthpb.HealthCheckResponse_SERVING}, nil
}

func (cs *checkoutService) Watch(req *healthpb.HealthCheckRequest, ws healthpb.Health_WatchServer) error {
	return status.Errorf(codes.Unimplemented, "health check via Watch not implemented")
}

func (cs *checkoutService) PlaceOrder(ctx context.Context, req *pb.PlaceOrderRequest) (*pb.PlaceOrderResponse, error) {
	placeOrderSpan := trace.SpanFromContext(ctx)
	// placeOrderSpan.SetAttributes(commonLabels...)
	defer placeOrderSpan.End()

	traceId := placeOrderSpan.SpanContext().TraceID()
	spanId := placeOrderSpan.SpanContext().SpanID()

	log.Infof("TraceID: %v SpanID: %v Start PlaceOrder user_id=%q user_currency=%q", traceId, spanId, req.UserId, req.UserCurrency)

	orderID, err := uuid.NewUUID()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate order uuid")
	}

	/*
	  Inject cpu consume fault, trigger by CheckoutPlaceOrderCPU flag
	*/
	if _, _err_ := failpoint.Eval(_curpkg_("CheckoutPlaceOrderCPU")); _err_ == nil {
		start := time.Now()
		for {
			// break for after duration
			if time.Now().Sub(start).Milliseconds() > 800 {
				break
			}
		}
	}

	log.Infof("TraceID: %v SpanID: %v Get OrderID %v", traceId, spanId, orderID)
	prep, err := cs.prepareOrderItemsAndShippingQuoteFromCart(ctx, req.UserId, req.UserCurrency, req.Address)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	total := pb.Money{CurrencyCode: req.UserCurrency,
		Units: 0,
		Nanos: 0}
	total = money.Must(money.Sum(total, *prep.shippingCostLocalized))
	for _, it := range prep.orderItems {
		multPrice := money.MultiplySlow(*it.Cost, uint32(it.GetItem().GetQuantity()))
		total = money.Must(money.Sum(total, multPrice))
	}

	txID, err := cs.chargeCard(ctx, &total, req.CreditCard)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to charge card: %+v", err)
	}
	log.Infof("TraceID: %v SpanID: %v Payment went through (transaction_id: %s)", traceId, spanId, txID)

	shippingTrackingID, err := cs.shipOrder(ctx, req.Address, prep.cartItems)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "shipping error: %+v", err)
	}

	_ = cs.emptyUserCart(ctx, req.UserId)

	orderResult := &pb.OrderResult{
		OrderId:            orderID.String(),
		ShippingTrackingId: shippingTrackingID,
		ShippingCost:       prep.shippingCostLocalized,
		ShippingAddress:    req.Address,
		Items:              prep.orderItems,
	}

	_ = cs.sendOrderConfirmation(ctx, req.Email, orderResult)

	resp := &pb.PlaceOrderResponse{Order: orderResult}

	log.Infof("TraceID: %v SpanID: %v PlaceOrder user_id=%q user_currency=%q successfully", traceId, spanId, req.UserId, req.UserCurrency)
	return resp, nil
}

type orderPrep struct {
	orderItems            []*pb.OrderItem
	cartItems             []*pb.CartItem
	shippingCostLocalized *pb.Money
}

func (cs *checkoutService) prepareOrderItemsAndShippingQuoteFromCart(ctx context.Context, userID, userCurrency string, address *pb.Address) (orderPrep, error) {
	ctx, prepareOrderItemsSpan := tracer.Start(ctx, "hipstershop.CheckoutService/PrepareOrderItemsAndShippingQuoteFromCart")
	defer prepareOrderItemsSpan.End()

	traceId := prepareOrderItemsSpan.SpanContext().TraceID()
	spanId := prepareOrderItemsSpan.SpanContext().SpanID()

	log.Infof("TraceID: %v SpanID: %v Start prepare order of user %v", traceId, spanId, userID)

	var out orderPrep
	cartItems, err := cs.getUserCart(ctx, userID)

	if err != nil {
		log.Errorf("TraceID: %v SpanID: %v Cart failure: %+v", traceId, spanId, err)
		return out, fmt.Errorf("cart failure: %+v", err)
	}
	orderItems, err := cs.prepOrderItems(ctx, cartItems, userCurrency)
	if err != nil {
		log.Errorf("TraceID: %v SpanID: %v Failed to prepare order: %+v", traceId, spanId, err)
		return out, fmt.Errorf("failed to prepare order: %+v", err)
	}
	shippingUSD, err := cs.quoteShipping(ctx, address, cartItems)

	if err != nil {
		log.Errorf("TraceID: %v SpanID: %v Shipping quote failure: %+v", traceId, spanId, err)
		return out, fmt.Errorf("shipping quote failure: %+v", err)
	}
	shippingPrice, err := cs.convertCurrency(ctx, shippingUSD, userCurrency)
	if err != nil {
		log.Errorf("TraceID: %v SpanID: %v Failed to convert shipping cost to currency: %+v", traceId, spanId, err)
		return out, fmt.Errorf("failed to convert shipping cost to currency: %+v", err)
	}

	out.shippingCostLocalized = shippingPrice
	out.cartItems = cartItems
	out.orderItems = orderItems

	log.Infof("TraceID: %v SpanID: %v Prepare order of user %v successfully", traceId, spanId, userID)
	return out, nil
}

func (cs *checkoutService) quoteShipping(ctx context.Context, address *pb.Address, items []*pb.CartItem) (*pb.Money, error) {
	ctx, quoteShippingSpan := tracer.Start(ctx, "hipstershop.CheckoutService/QuoteShipping")
	defer quoteShippingSpan.End()

	traceId := quoteShippingSpan.SpanContext().TraceID()
	spanId := quoteShippingSpan.SpanContext().SpanID()

	log.Infof("TraceID: %v SpanID: %v Quote shipping with items %v", traceId, spanId, items)

	conn, err := grpc.DialContext(ctx, cs.shippingSvcAddr,
		grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(otelgrpc.UnaryClientInterceptor()),
		grpc.WithStreamInterceptor(otelgrpc.StreamClientInterceptor()),
	)
	if err != nil {
		log.Errorf("TraceID: %v SpanID: %v Could not connect shipping service: %+v", traceId, spanId, err)
		return nil, fmt.Errorf("could not connect shipping service: %+v", err)
	}
	defer conn.Close()

	shippingQuote, err := pb.NewShippingServiceClient(conn).
		GetQuote(ctx, &pb.GetQuoteRequest{
			Address: address,
			Items:   items})
	if err != nil {
		log.Errorf("TraceID: %v SpanID: %v Failed to get shipping quote: %+v", traceId, spanId, err)
		return nil, fmt.Errorf("failed to get shipping quote: %+v", err)
	}

	cost := shippingQuote.GetCostUsd()
	log.Infof("TraceID: %v SpanID: %v Get shipping cost %v", traceId, spanId, cost)
	return cost, nil
}

func (cs *checkoutService) getUserCart(ctx context.Context, userID string) ([]*pb.CartItem, error) {
	ctx, getUserCartSpan := tracer.Start(ctx, "hipstershop.CheckoutService/GetUserCart")
	defer getUserCartSpan.End()

	traceId := getUserCartSpan.SpanContext().TraceID()
	spanId := getUserCartSpan.SpanContext().SpanID()

	log.Infof("TraceID: %v SpanID: %v Query cart of user %v", traceId, spanId, userID)

	conn, err := grpc.DialContext(ctx, cs.cartSvcAddr, grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(otelgrpc.UnaryClientInterceptor()),
		grpc.WithStreamInterceptor(otelgrpc.StreamClientInterceptor()),
	)
	if err != nil {
		log.Errorf("TraceID: %v SpanID: %v Could not connect cart service: %+v", traceId, spanId, err)
		return nil, fmt.Errorf("could not connect cart service: %+v", err)
	}
	defer conn.Close()

	cart, err := pb.NewCartServiceClient(conn).GetCart(ctx, &pb.GetCartRequest{UserId: userID})

	if err != nil {
		log.Errorf("TraceID: %v SpanID: %v Failed to get user cart during checkout: %+v", traceId, spanId, err)
		return nil, fmt.Errorf("failed to get user cart during checkout: %+v", err)
	}
	log.Infof("TraceID: %v SpanID: %v Get user %v cart %v", traceId, spanId, userID, cart)
	return cart.GetItems(), nil
}

func (cs *checkoutService) emptyUserCart(ctx context.Context, userID string) error {
	ctx, emptyUserCartSpan := tracer.Start(ctx, "hipstershop.CheckoutService/EmptyUserCart")
	defer emptyUserCartSpan.End()

	traceId := emptyUserCartSpan.SpanContext().TraceID()
	spanId := emptyUserCartSpan.SpanContext().SpanID()

	log.Infof("TraceID: %v SpanID: %v Start empty the cart of user %v", traceId, spanId, userID)

	conn, err := grpc.DialContext(ctx, cs.cartSvcAddr,
		grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(otelgrpc.UnaryClientInterceptor()),
		grpc.WithStreamInterceptor(otelgrpc.StreamClientInterceptor()),
	)
	if err != nil {
		log.Errorf("TraceID: %v SpanID: %v Could not connect cart service: %+v", traceId, spanId, err)
		return fmt.Errorf("could not connect cart service: %+v", err)
	}
	defer conn.Close()

	_, err = pb.NewCartServiceClient(conn).EmptyCart(ctx, &pb.EmptyCartRequest{UserId: userID})

	if err != nil {
		log.Errorf("TraceID: %v SpanID: %v Failed to empty user cart during checkout: %+v", traceId, spanId, err)
		return fmt.Errorf("failed to empty user cart during checkout: %+v", err)
	}
	log.Infof("TraceID: %v SpanID: %v Empty the cart of user %v successfully", traceId, spanId, userID)
	return nil
}

func (cs *checkoutService) prepOrderItems(ctx context.Context, items []*pb.CartItem, userCurrency string) ([]*pb.OrderItem, error) {
	ctx, prepOrderItemsSpan := tracer.Start(ctx, "hipstershop.CheckoutService/PrepOrderItems")
	defer prepOrderItemsSpan.End()

	traceId := prepOrderItemsSpan.SpanContext().TraceID()
	spanId := prepOrderItemsSpan.SpanContext().SpanID()

	log.Infof("TraceID: %v SpanID: %v Start query cost of products %v ", traceId, spanId, items)

	out := make([]*pb.OrderItem, len(items))

	conn, err := grpc.DialContext(ctx, cs.productCatalogSvcAddr,
		grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(otelgrpc.UnaryClientInterceptor()),
		grpc.WithStreamInterceptor(otelgrpc.StreamClientInterceptor()),
	)
	if err != nil {
		log.Errorf("TraceID: %v SpanID: %v Could not connect product catalog service: %+v", traceId, spanId, err)
		return nil, fmt.Errorf("could not connect product catalog service: %+v", err)
	}
	defer conn.Close()
	cl := pb.NewProductCatalogServiceClient(conn)

	for i, item := range items {
		product, err := cl.GetProduct(ctx, &pb.GetProductRequest{Id: item.GetProductId()})
		if err != nil {
			log.Errorf("TraceID: %v SpanID: %v Failed to get product #%q", traceId, spanId, item.GetProductId())
			return nil, fmt.Errorf("failed to get product #%q", item.GetProductId())
		}
		price, err := cs.convertCurrency(ctx, product.GetPriceUsd(), userCurrency)
		if err != nil {
			log.Errorf("TraceID: %v SpanID: %v Failed to convert price of %q to %s", traceId, spanId, item.GetProductId(), userCurrency)
			return nil, fmt.Errorf("failed to convert price of %q to %s", item.GetProductId(), userCurrency)
		}
		out[i] = &pb.OrderItem{
			Item: item,
			Cost: price}
	}

	log.Infof("TraceID: %v SpanID: %v Query cost of products %v succefully", traceId, spanId, items)

	return out, nil
}

func (cs *checkoutService) convertCurrency(ctx context.Context, from *pb.Money, toCurrency string) (*pb.Money, error) {
	ctx, convertCurrencySpan := tracer.Start(ctx, "hipstershop.CheckoutService/ConvertCurrency")
	defer convertCurrencySpan.End()

	traceId := convertCurrencySpan.SpanContext().TraceID()
	spanId := convertCurrencySpan.SpanContext().SpanID()

	log.Infof("TraceID: %v SpanID: %v Start convert currency from %v to %v ", traceId, spanId, from, toCurrency)
	conn, err := grpc.DialContext(ctx, cs.currencySvcAddr,
		grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(otelgrpc.UnaryClientInterceptor()),
		grpc.WithStreamInterceptor(otelgrpc.StreamClientInterceptor()),
	)
	if err != nil {
		log.Errorf("TraceID: %v SpanID: %v Could not connect currency service: %+v", traceId, spanId, err)
		return nil, fmt.Errorf("could not connect currency service: %+v", err)
	}
	defer conn.Close()
	result, err := pb.NewCurrencyServiceClient(conn).Convert(context.TODO(), &pb.CurrencyConversionRequest{
		From:   from,
		ToCode: toCurrency})

	if err != nil {
		log.Errorf("TraceID: %v SpanID: %v Failed to convert currency: %+v", traceId, spanId, err)
		return nil, fmt.Errorf("failed to convert currency: %+v", err)
	}
	log.Infof("TraceID: %v SpanID: %v Convert currency from %v to %v successfully", traceId, spanId, from, toCurrency)
	return result, err
}

func (cs *checkoutService) chargeCard(ctx context.Context, amount *pb.Money, paymentInfo *pb.CreditCardInfo) (string, error) {
	ctx, chargeCardSpan := tracer.Start(ctx, "hipstershop.CheckoutService/ChargeCard")
	defer chargeCardSpan.End()

	traceId := chargeCardSpan.SpanContext().TraceID()
	spanId := chargeCardSpan.SpanContext().SpanID()

	log.Infof("TraceID: %v SpanID: %v Start charge card", traceId, spanId)
	conn, err := grpc.DialContext(ctx, cs.paymentSvcAddr,
		grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(otelgrpc.UnaryClientInterceptor()),
		grpc.WithStreamInterceptor(otelgrpc.StreamClientInterceptor()),
	)

	/*
		Inject Exception Fault, trigger by CheckoutChargeCardException flag
	*/
	if _, _err_ := failpoint.Eval(_curpkg_("CheckoutChargeCardException")); _err_ == nil {
		err = fmt.Errorf("connect error.")
	}

	if err != nil {
		log.Errorf("TraceID: %v SpanID: %v Failed to connect payment service: %+v", traceId, spanId, err)
		return "", fmt.Errorf("failed to connect payment service: %+v", err)
	}
	defer conn.Close()

	paymentResp, err := pb.NewPaymentServiceClient(conn).Charge(ctx, &pb.ChargeRequest{
		Amount:     amount,
		CreditCard: paymentInfo})

	/*
	 Inject modify return vaule Fault, trigger by CheckoutChargeCardReturn flag
	*/
	if _, _err_ := failpoint.Eval(_curpkg_("CheckoutChargeCardReturn")); _err_ == nil {
		err = fmt.Errorf("charge card error.")
	}

	if err != nil {
		log.Errorf("TraceID: %v SpanID: %v Could not charge the card: %+v", traceId, spanId, err)
		return "", fmt.Errorf("could not charge the card: %+v", err)
	}
	log.Infof("TraceID: %v SpanID: %v Charge successfully ", traceId, spanId)
	return paymentResp.GetTransactionId(), nil
}

func (cs *checkoutService) sendOrderConfirmation(ctx context.Context, email string, order *pb.OrderResult) error {
	ctx, sendOrderConfirmationSpan := tracer.Start(ctx, "hipstershop.CheckoutService/SendOrderConfirmation")
	defer sendOrderConfirmationSpan.End()

	traceId := sendOrderConfirmationSpan.SpanContext().TraceID()
	spanId := sendOrderConfirmationSpan.SpanContext().SpanID()

	log.Infof("TraceID: %v SpanID: %v Start send Order Confirmation to email %v", traceId, spanId, email)
	conn, err := grpc.DialContext(ctx, cs.emailSvcAddr,
		grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(otelgrpc.UnaryClientInterceptor()),
		grpc.WithStreamInterceptor(otelgrpc.StreamClientInterceptor()),
	)

	if err != nil {
		log.Errorf("TraceID: %v SpanID: %v Failed to connect email service: %+v", traceId, spanId, err)
		return fmt.Errorf("failed to connect email service: %+v", err)
	}

	defer conn.Close()
	_, err = pb.NewEmailServiceClient(conn).SendOrderConfirmation(ctx, &pb.SendOrderConfirmationRequest{
		Email: email,
		Order: order})

	if err != nil {
		log.Warnf("TraceID: %v SpanID: %v Failed to send order confirmation to %q: %+v", traceId, spanId, email, err)
	} else {
		log.Infof("TraceID: %v SpanID: %v Order confirmation email sent to %q", traceId, spanId, email)
	}
	return err
}

func (cs *checkoutService) shipOrder(ctx context.Context, address *pb.Address, items []*pb.CartItem) (string, error) {
	ctx, shipOrderSpan := tracer.Start(ctx, "hipstershop.CheckoutService/ShipOrder")
	defer shipOrderSpan.End()

	traceId := shipOrderSpan.SpanContext().TraceID()
	spanId := shipOrderSpan.SpanContext().SpanID()

	log.Infof("TraceID: %v SpanID: %v Start ship Order to address %v", traceId, spanId, address)

	conn, err := grpc.DialContext(ctx, cs.shippingSvcAddr,
		grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(otelgrpc.UnaryClientInterceptor()),
		grpc.WithStreamInterceptor(otelgrpc.StreamClientInterceptor()),
	)

	if err != nil {
		log.Errorf("TraceID: %v SpanID: %v Failed to connect email service: %+v", traceId, spanId, err)
		return "", fmt.Errorf("failed to connect email service: %+v", err)
	}
	defer conn.Close()
	resp, err := pb.NewShippingServiceClient(conn).ShipOrder(ctx, &pb.ShipOrderRequest{
		Address: address,
		Items:   items})
	if err != nil {
		log.Errorf("TraceID: %v SpanID: %v Shipment failed: %+v", traceId, spanId, err)
		return "", fmt.Errorf("shipment failed: %+v", err)
	}

	log.Infof("TraceID: %v SpanID: %v Ship Order to address %v successfully", traceId, spanId, address)
	return resp.GetTrackingId(), nil
}

// TODO: Dial and create client once, reuse.
