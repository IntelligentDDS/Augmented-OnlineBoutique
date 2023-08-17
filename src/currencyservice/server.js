/*
 * Copyright 2018 Google LLC.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

const SERVICE_NAME = process.env.SERVICE_NAME;
const POD_NAME = process.env.POD_NAME;
const NODE_NAME = process.env.NODE_NAME;

const api = require('@opentelemetry/api');
const tracer = require('./tracer')((SERVICE_NAME));

const path = require('path');
const grpc = require('grpc');
const pino = require('pino');
const protoLoader = require('@grpc/proto-loader');

const MAIN_PROTO_PATH = path.join(__dirname, './proto/demo.proto');
const HEALTH_PROTO_PATH = path.join(__dirname, './proto/grpc/health/v1/health.proto');

const PORT = process.env.PORT;

const shopProto = _loadProto(MAIN_PROTO_PATH).hipstershop;
const healthProto = _loadProto(HEALTH_PROTO_PATH).grpc.health.v1;

const logger = pino({
  name: 'currencyservice-server',
  messageKey: 'message',
  timestamp: false, //pino.stdTimeFunctions.isoTime,
  changeLevelName: 'severity',
  useLevelLabels: true
});

/**
 * Helper function that loads a protobuf file.
 */
function _loadProto(path) {
  const packageDefinition = protoLoader.loadSync(
    path,
    {
      keepCase: true,
      longs: String,
      enums: String,
      defaults: true,
      oneofs: true
    }
  );
  return grpc.loadPackageDefinition(packageDefinition);
}

/**
 * Helper function that gets currency data from a stored JSON file
 * Uses public data from European Central Bank
 */
function _getCurrencyData(callback) {
  const parentSpan = api.trace.getSpan(api.context.active());

  const currentSpan = tracer.startSpan('hipstershop.CurrencyService/GetCurrencyData', {
    parent: parentSpan,
    attributes: {
      "PodName": POD_NAME,
      "NodeName": NODE_NAME,
    },
  });
  const spanId = currentSpan.spanContext().spanId;
  const traceId = currentSpan.spanContext().traceId;
  currentSpan.setAttribute("PodName", POD_NAME);
  currentSpan.setAttribute("NodeName", NODE_NAME);
  logger.info(`TraceID: ${traceId} SpanID: ${spanId} Getting currency data`);

  const data = require('./data/currency_conversion.json');
  callback(data);
  logger.info(`TraceID: ${traceId} SpanID: ${spanId} Get currency data successful`);
  currentSpan.end();
}

/**
 * Helper function that handles decimal/fractional carrying
 */
function _carry(amount) {
  const parentSpan = api.trace.getSpan(api.context.active());

  const currentSpan = tracer.startSpan('hipstershop.CurrencyService/Carry', {
    parent: parentSpan,
    attributes: {
      "PodName": POD_NAME,
      "NodeName": NODE_NAME,
    },
  });
  const spanId = currentSpan.spanContext().spanId;
  const traceId = currentSpan.spanContext().traceId;
  currentSpan.setAttribute("PodName", POD_NAME);
  currentSpan.setAttribute("NodeName", NODE_NAME);
  logger.info(`TraceID: ${traceId} SpanID: ${spanId} Handles decimal or fractional carrying`);

  const fractionSize = Math.pow(10, 9);
  amount.nanos += (amount.units % 1) * fractionSize;
  amount.units = Math.floor(amount.units) + Math.floor(amount.nanos / fractionSize);
  amount.nanos = amount.nanos % fractionSize;
  currentSpan.end();
  return amount;
}

/**
 * Lists the supported currencies
 */
function getSupportedCurrencies(call, callback) {
  const currentSpan = api.trace.getSpan(api.context.active());
  const spanId = currentSpan.spanContext().spanId;
  const traceId = currentSpan.spanContext().traceId;
  currentSpan.setAttribute("PodName", POD_NAME);
  currentSpan.setAttribute("NodeName", NODE_NAME);
  logger.info(`TraceID: ${traceId} SpanID: ${spanId} Getting supported currencies...`);
  _getCurrencyData((data) => {
    callback(null, { currency_codes: Object.keys(data) });
  });
}

/**
 * Converts between currencies
 */
function convert(call, callback) {
  const currentSpan = api.trace.getSpan(api.context.active());
  const spanId = currentSpan.spanContext().spanId;
  const traceId = currentSpan.spanContext().traceId;
  currentSpan.setAttribute("PodName", POD_NAME);
  currentSpan.setAttribute("NodeName", NODE_NAME);
  logger.info(`TraceID: ${traceId} SpanID: ${spanId} Received conversion request`);
  try {
    _getCurrencyData((data) => {
      const request = call.request;

      // Convert: from_currency --> EUR
      const from = request.from;
      const euros = _carry({
        units: from.units / data[from.currency_code],
        nanos: from.nanos / data[from.currency_code]
      });

      euros.nanos = Math.round(euros.nanos);

      // Convert: EUR --> to_currency
      const result = _carry({
        units: euros.units * data[request.to_code],
        nanos: euros.nanos * data[request.to_code]
      });

      result.units = Math.floor(result.units);
      result.nanos = Math.floor(result.nanos);
      result.currency_code = request.to_code;

      logger.info(`TraceID: ${traceId} SpanID: ${spanId} Conversion request successful`);
      callback(null, result);
    });
  } catch (err) {
    logger.error(`TraceID: ${traceId} SpanID: ${spanId} Conversion request failed: ${err}`);
    callback(err.message);
  }
}

/**
 * Endpoint for health checks
 */
function check(call, callback) {
  callback(null, { status: 'SERVING' });
}

/**
 * Starts an RPC server that receives requests for the
 * CurrencyConverter service at the sample server port
 */
function main() {
  logger.info(`Starting gRPC server on port ${PORT}...`);
  const server = new grpc.Server();
  server.addService(shopProto.CurrencyService.service, { getSupportedCurrencies, convert });
  server.addService(healthProto.Health.service, { check });
  server.bind(`0.0.0.0:${PORT}`, grpc.ServerCredentials.createInsecure());
  server.start();
}

main();
