#!/usr/bin/python
#
# Copyright 2018 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import os
import random
import time
import traceback
from concurrent import futures

import grpc
import demo_pb2
import demo_pb2_grpc
from grpc_health.v1 import health_pb2
from grpc_health.v1 import health_pb2_grpc

from opentelemetry import trace
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import (
    OTLPSpanExporter,
)
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.sdk.resources import SERVICE_NAME, Resource
from opentelemetry.instrumentation.grpc import GrpcInstrumentorServer
from opentelemetry.instrumentation.grpc.grpcext import intercept_channel
from opentelemetry.instrumentation.grpc import client_interceptor

from logger import getJSONLogger
logger = getJSONLogger('recommendationservice-server')

span_exporter = OTLPSpanExporter(
    # optional
    endpoint=os.environ.get('COLLECTOR_ADDR'),
    # credentials=ChannelCredentials(credentials),
    # headers=(("metadata", "metadata")),
)

trace.set_tracer_provider(TracerProvider(resource=Resource.create({SERVICE_NAME: os.environ.get('SERVICE_NAME'), 
                                                                  "NodeName": os.environ.get('NODE_NAME'),
                                                                  "PodName": os.environ.get('POD_NAME')})))
trace.get_tracer_provider().add_span_processor(
    BatchSpanProcessor(span_exporter)
)

# Configure the tracer to use the collector exporter
tracer = trace.get_tracer_provider().get_tracer(__name__)
grpc_server_instrumentor = GrpcInstrumentorServer()
grpc_server_instrumentor.instrument()


class RecommendationService(demo_pb2_grpc.RecommendationServiceServicer):
    def ListRecommendations(self, request, context):
        span = trace.get_current_span()
        trace_id = format(span.get_span_context().trace_id, "032x")
        span_id = format(span.get_span_context().span_id, "016x")
        max_responses = 5
        try:
            logger.info("TraceID: {} SpanID: {} Query Recommendations".format(trace_id, span_id))
            cat_response = product_catalog_stub.ListProducts(demo_pb2.Empty())
            product_ids = [x.id for x in cat_response.products]
            filtered_products = list(set(product_ids)-set(request.product_ids))
            num_products = len(filtered_products)
            num_return = min(max_responses, num_products)
            indices = random.sample(range(num_products), num_return)
            prod_list = [filtered_products[i] for i in indices]
            logger.info("TraceID: {} SpanID: {} List Recommendations product_ids={}".format(trace_id, span_id, prod_list))
            response = demo_pb2.ListRecommendationsResponse()
            response.product_ids.extend(prod_list)
            return response
        except Exception as err:
            logger.error("TraceID: {} SpanID: {} Failed to list recommendations with error={}".format(trace_id, span_id, err))
            return demo_pb2.Empty()
            
    def Check(self, request, context):
        return health_pb2.HealthCheckResponse(
            status=health_pb2.HealthCheckResponse.SERVING)

    def Watch(self, request, context):
        return health_pb2.HealthCheckResponse(
            status=health_pb2.HealthCheckResponse.UNIMPLEMENTED)


if __name__ == "__main__":
    logger.info("initializing recommendationservice")

    port = os.environ.get('PORT', "8080")
    catalog_addr = os.environ.get('PRODUCT_CATALOG_SERVICE_ADDR', '')
    if catalog_addr == "":
        raise Exception('PRODUCT_CATALOG_SERVICE_ADDR environment variable not set')
    logger.info("product catalog address: " + catalog_addr)
    channel = grpc.insecure_channel(catalog_addr)
    channel = intercept_channel(channel, client_interceptor())
    product_catalog_stub = demo_pb2_grpc.ProductCatalogServiceStub(channel)

    # create gRPC server
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))

    # add class to gRPC server
    service = RecommendationService()
    demo_pb2_grpc.add_RecommendationServiceServicer_to_server(service, server)
    health_pb2_grpc.add_HealthServicer_to_server(service, server)

    # start server
    logger.info("listening on port: " + port)
    server.add_insecure_port('[::]:'+port)
    server.start()

    # keep alive
    try:
         while True:
            time.sleep(10000)
    except KeyboardInterrupt:
            server.stop(0)
