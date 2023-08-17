package hipstershop;

import io.opentelemetry.api.OpenTelemetry;
import io.opentelemetry.exporter.otlp.trace.OtlpGrpcSpanExporter;
import io.opentelemetry.sdk.OpenTelemetrySdk;
import io.opentelemetry.sdk.autoconfigure.OpenTelemetryResourceAutoConfiguration;
import io.opentelemetry.sdk.trace.SdkTracerProvider;
import io.opentelemetry.sdk.trace.export.BatchSpanProcessor;
import io.opentelemetry.sdk.trace.samplers.Sampler;
import java.util.Collections;
import java.util.concurrent.TimeUnit;

class TraceConfiguration {
  // private static final String smapleRatio = System.getenv("SAMPLE_RATIO");
  private static final String endpoint = System.getenv("OTEL_EXPORTER_OTLP_ENDPOINT");

  /**
   * Adds a BatchSpanProcessor initialized with OtlpGrpcSpanExporter to the TracerSdkProvider.
   *
   * @return a ready-to-use {@link OpenTelemetry} instance.
   */
  static OpenTelemetry initOpenTelemetry() {
    OtlpGrpcSpanExporter spanExporter = OtlpGrpcSpanExporter.builder()
                                            .setEndpoint(endpoint)
                                            .setTimeout(30, TimeUnit.SECONDS)
                                            .build();

    BatchSpanProcessor spanProcessor = BatchSpanProcessor.builder(spanExporter)
                                           .setScheduleDelay(100, TimeUnit.MILLISECONDS)
                                           .build();
                                    
    SdkTracerProvider tracerProvider = SdkTracerProvider.builder()
                                           .addSpanProcessor(spanProcessor)
                                           .setSampler(Sampler.alwaysOn())
                                           .setResource(OpenTelemetryResourceAutoConfiguration.configureResource())
                                           .build();
    
    OpenTelemetrySdk openTelemetrySdk = OpenTelemetrySdk.builder().setTracerProvider(tracerProvider).buildAndRegisterGlobal();

    Runtime.getRuntime().addShutdownHook(new Thread(tracerProvider::close));

    return openTelemetrySdk;
  }
}