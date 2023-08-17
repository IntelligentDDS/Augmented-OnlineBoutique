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

from concurrent import futures
import argparse
import os
import sys
import time
import grpc
import traceback
from jinja2 import Environment, FileSystemLoader, select_autoescape, TemplateError
from google.api_core.exceptions import GoogleAPICallError
from google.auth.exceptions import DefaultCredentialsError

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

from logger import getJSONLogger
logger = getJSONLogger('emailservice-server')

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

# try:
#     googleclouddebugger.enable(
#         module='emailserver',
#         version='1.0.0'
#     )
# except:
#     pass

# Loads confirmation email template from file
env = Environment(
    loader=FileSystemLoader('templates'),
    autoescape=select_autoescape(['html', 'xml'])
)
template = env.get_template('confirmation.html')

class BaseEmailService(demo_pb2_grpc.EmailServiceServicer):
  def Check(self, request, context):
    return health_pb2.HealthCheckResponse(
      status=health_pb2.HealthCheckResponse.SERVING)
  
  def Watch(self, request, context):
    return health_pb2.HealthCheckResponse(
      status=health_pb2.HealthCheckResponse.UNIMPLEMENTED)

class EmailService(BaseEmailService):
  def __init__(self):
    raise Exception('cloud mail client not implemented')
    super().__init__()

  @staticmethod
  def send_email(client, email_address, content):
    response = client.send_message(
      sender = client.sender_path(project_id, region, sender_id),
      envelope_from_authority = '',
      header_from_authority = '',
      envelope_from_address = from_address,
      simple_message = {
        "from": {
          "address_spec": from_address,
        },
        "to": [{
          "address_spec": email_address
        }],
        "subject": "Your Confirmation Email",
        "html_body": content
      }
    )
    logger.info("Message sent: {}".format(response.rfc822_message_id))

  def SendOrderConfirmation(self, request, context):
    email = request.email
    order = request.order

    try:
      confirmation = template.render(order = order)
    except TemplateError as err:
      context.set_details("An error occurred when preparing the confirmation mail.")
      logger.error(err.message)
      context.set_code(grpc.StatusCode.INTERNAL)
      return demo_pb2.Empty()

    try:
      EmailService.send_email(self.client, email, confirmation)
    except GoogleAPICallError as err:
      context.set_details("An error occurred when sending the email.")
      print(err.message)
      context.set_code(grpc.StatusCode.INTERNAL)
      return demo_pb2.Empty()
    return demo_pb2.Empty()

class DummyEmailService(BaseEmailService):
  def SendOrderConfirmation(self, request, context):
    span = trace.get_current_span()
    # span.set_attribute("NodeName", os.environ.get('NODE_NAME'))
    # span.set_attribute("PodName", os.environ.get('POD_NAME'))
    trace_id = format(span.get_span_context().trace_id, "032x")
    span_id = format(span.get_span_context().span_id, "016x")

    email = request.email
    order = request.order
    logger.info('TraceID: {} SpanID: {} Send order confirmation email to {} start.'.format(trace_id, span_id,email))
    try:
      confirmation = template.render(order = order)
    except Exception as err:
      context.set_details("An error occurred when preparing the confirmation mail.")
      logger.error('TraceID: {} SpanID: {} Failed to send order confirmation email to {}. Error = {}.'.format(trace_id, span_id,request.email, err))
      context.set_code(grpc.StatusCode.INTERNAL)
      return demo_pb2.Empty()

    logger.info('TraceID: {} SpanID: {} A request to send order confirmation email to {} has been received.'.format(trace_id, span_id,request.email))
    return demo_pb2.Empty()

class HealthCheck():
  def Check(self, request, context):
    return health_pb2.HealthCheckResponse(
      status=health_pb2.HealthCheckResponse.SERVING)

def start(dummy_mode):
  server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
  service = None
  if dummy_mode:
    service = DummyEmailService()
  else:
    raise Exception('non-dummy mode not implemented yet')

  demo_pb2_grpc.add_EmailServiceServicer_to_server(service, server)
  health_pb2_grpc.add_HealthServicer_to_server(service, server)

  port = os.environ.get('PORT', "8080")
  logger.info("listening on port: "+port)
  server.add_insecure_port('[::]:'+port)
  server.start()
  try:
    while True:
      time.sleep(3600)
  except KeyboardInterrupt:
    server.stop(0)

if __name__ == '__main__':
  logger.info('starting the email service in dummy mode.')

  start(dummy_mode = True)
