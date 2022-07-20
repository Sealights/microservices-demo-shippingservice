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
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	pb "github.com/GoogleCloudPlatform/microservices-demo/src/shippingservice/genproto/hipstershop"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

const (
	defaultPort     = "50051"
	defaultHttpPort = "50052"
)

var log *logrus.Logger

func init() {
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
}

func InitTracerProvider() *sdktrace.TracerProvider {
	ctx := context.Background()

	collectorUrl := os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT")
	token := os.Getenv("RM_DEV_SL_TOKEN")

	if token == "" {
		log.Fatal(fmt.Errorf("empty token"))
	}

	protocol := os.Getenv("OTEL_AGENT_COLLECTOR_PROTOCOL")

	if protocol == "" {
		log.Fatal(fmt.Errorf("empty protocol"))
	}

	if collectorUrl == "" {
		var err error
		collectorUrl, err = decodeAndExtractServerUrl(token, os.Getenv("OTEL_AGENT_COLLECTOR_PORT"))
		if err != nil {
			log.Fatal(err)
		}
	}

	headers := make(map[string]string)
	headers["Authorization"] = "Bearer " + token
	headers["x-otlp-protocol"] = protocol

	var traceOptions = []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(collectorUrl),
		otlptracegrpc.WithHeaders(headers)}

	exporter, err := otlptracegrpc.New(ctx, traceOptions...)
	if err != nil {
		log.Fatal(err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(exporter),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	return tp
}

func main() {
	tp := InitTracerProvider()
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}()

	port := defaultPort
	if value, ok := os.LookupEnv("PORT"); ok {
		port = value
	}
	port = fmt.Sprintf(":%s", port)

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	var srv *grpc.Server = grpc.NewServer(
		grpc.UnaryInterceptor(otelgrpc.UnaryServerInterceptor()),
		grpc.StreamInterceptor(otelgrpc.StreamServerInterceptor()),
	)

	svc := &server{}
	pb.RegisterShippingServiceServer(srv, svc)
	healthpb.RegisterHealthServer(srv, svc)
	log.Infof("Shipping Service listening on port %s", port)

	// Register reflection service on gRPC server.
	reflection.Register(srv)

	go RunGrpcServer(srv, lis)
	RunHttpServer()
}

func RunGrpcServer(srv *grpc.Server, lis net.Listener) {
	err := srv.Serve(lis)
	if err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func RunHttpServer() {
	httpPort := defaultHttpPort

	if value, ok := os.LookupEnv("HTTP_PORT"); ok {
		httpPort = value
	}

	httpPort = fmt.Sprintf(":%s", httpPort)
	http.HandleFunc("/getquote", GetQuote)
	http.HandleFunc("/shiporder", ShipOrder)

	log.Infof("Shipping Service Http starting on port %s", httpPort)

	err := http.ListenAndServe(fmt.Sprintf("%s", httpPort), nil)
	if err != nil {
		log.Fatalf("failed to http serve: %v", err)
	}
}

// server controls RPC service responses.
type server struct {
	pb.UnimplementedShippingServiceServer
}

// Check is for health checking.
func (s *server) Check(ctx context.Context, req *healthpb.HealthCheckRequest) (*healthpb.HealthCheckResponse, error) {
	return &healthpb.HealthCheckResponse{Status: healthpb.HealthCheckResponse_SERVING}, nil
}

func (s *server) Watch(req *healthpb.HealthCheckRequest, ws healthpb.Health_WatchServer) error {
	return status.Errorf(codes.Unimplemented, "health check via Watch not implemented")
}

// GetQuote produces a shipping quote (cost) in USD.
func (s *server) GetQuote(ctx context.Context, in *pb.GetQuoteRequest) (*pb.GetQuoteResponse, error) {
	log.Info("[GetQuote] received request")
	defer log.Info("[GetQuote] completed request")

	// 1. Our quote system requires the total number of items to be shipped.
	count := 0
	for _, item := range in.Items {
		count += int(item.Quantity)
	}

	// 2. Generate a quote based on the total number of items to be shipped.
	quote := CreateQuoteFromCount(count)

	// 3. Generate a response.
	return &pb.GetQuoteResponse{
		CostUsd: &pb.Money{
			CurrencyCode: "USD",
			Units:        int64(quote.Dollars),
			Nanos:        int32(quote.Cents * 10000000)},
	}, nil

}

// ShipOrder mocks that the requested items will be shipped.
// It supplies a tracking ID for notional lookup of shipment delivery status.
func (s *server) ShipOrder(ctx context.Context, in *pb.ShipOrderRequest) (*pb.ShipOrderResponse, error) {

	log.Info("[ShipOrder] received request")
	defer log.Info("[ShipOrder] completed request")
	// 1. Create a Tracking ID
	baseAddress := fmt.Sprintf("%s, %s, %s", in.Address.StreetAddress, in.Address.City, in.Address.State)
	id := CreateTrackingId(baseAddress)

	// 2. Generate a response.
	return &pb.ShipOrderResponse{
		TrackingId: id,
	}, nil
}

func decodeAndExtractServerUrl(token string, port string) (string, error) {
	if token == "" {
		return "", fmt.Errorf("token not found")
	}

	claims := jwt.MapClaims{}
	keyFunc := func(token *jwt.Token) (interface{}, error) {
		return []byte(""), nil
	}

	jwt.ParseWithClaims(token, claims, keyFunc)

	urlValue := claims["x-sl-server"].(string)
	if urlValue == "" {
		return "", fmt.Errorf("empty url server value")
	}

	host := strings.Trim(urlValue, "https://")
	host = strings.Trim(host, "/api")

	return fmt.Sprintf("ingest.%s:%s", host, port), nil
}
