'use strict';

const opentelemetry = require('@opentelemetry/api');
const { registerInstrumentations } = require('@opentelemetry/instrumentation');
const { NodeTracerProvider } = require('@opentelemetry/sdk-trace-node');
const { Resource } = require('@opentelemetry/resources');
const { SemanticResourceAttributes } = require('@opentelemetry/semantic-conventions');
const { SimpleSpanProcessor } = require('@opentelemetry/sdk-trace-base');
const { GrpcInstrumentation } = require('@opentelemetry/instrumentation-grpc');
const { CollectorTraceExporter } = require('@opentelemetry/exporter-collector-grpc');

const COLLECTOR_ADDR = "grpc://" + process.env.COLLECTOR_ADDR;

const collectorOptions = {
    // url is optional and can be omitted - default is grpc://localhost:4317
    url: COLLECTOR_ADDR,
};

module.exports = (serviceName) => {
    const provider = new NodeTracerProvider({
        resource: new Resource({
            [SemanticResourceAttributes.SERVICE_NAME]: serviceName,
        }),
    });

    const exporter = new CollectorTraceExporter(collectorOptions);

    provider.addSpanProcessor(new SimpleSpanProcessor(exporter));

    // Initialize the OpenTelemetry APIs to use the NodeTracerProvider bindings
    provider.register();

    registerInstrumentations({
        instrumentations: [
            new GrpcInstrumentation(),
        ],
    });

    return opentelemetry.trace.getTracer('currency-tracer');
};

