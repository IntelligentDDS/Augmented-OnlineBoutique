/*
 * Copyright 2018, Google LLC.
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

package hipstershop;

import hipstershop.Demo.Ad;
import hipstershop.Demo.AdRequest;
import hipstershop.Demo.AdResponse;
import io.grpc.CallOptions;
import io.grpc.Channel;
import io.grpc.ClientCall;
import io.grpc.ClientInterceptor;
import io.grpc.ForwardingClientCall;
import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;
import io.grpc.Metadata;
import io.grpc.MethodDescriptor;
import io.grpc.StatusRuntimeException;
import java.util.concurrent.TimeUnit;
import javax.annotation.Nullable;
import org.apache.logging.log4j.Level;
import org.apache.logging.log4j.LogManager;
import org.apache.logging.log4j.Logger;
import io.opentelemetry.api.OpenTelemetry;
import io.opentelemetry.api.trace.Span;
import io.opentelemetry.api.trace.SpanKind;
import io.opentelemetry.api.trace.SpanContext;
import io.opentelemetry.api.trace.StatusCode;
import io.opentelemetry.api.trace.Tracer;
import io.opentelemetry.context.Context;
import io.opentelemetry.context.Scope;
import io.opentelemetry.context.propagation.TextMapPropagator;
import io.opentelemetry.context.propagation.TextMapSetter;

/** A simple client that requests ads from the Ads Service. */
public class AdServiceClient {
  private static final Logger logger = LogManager.getLogger(AdServiceClient.class);

  private final ManagedChannel channel;
  private final hipstershop.AdServiceGrpc.AdServiceBlockingStub blockingStub;

  private static final String podName = System.getenv("POD_NAME");
  private static final String nodeName = System.getenv("NODE_NAME");

  private final String serverHostname;
  private final Integer serverPort;

  // it is important to initialize the OpenTelemetry SDK as early as possible in your application's
  // lifecycle.
  private static final OpenTelemetry openTelemetry = TraceConfiguration.initOpenTelemetry();

  // OTel Tracing API
  private final Tracer tracer =
      openTelemetry.getTracer("hipstershop.AdServiceClient");
  // Share context via text headers
  private final TextMapPropagator textFormat =
      openTelemetry.getPropagators().getTextMapPropagator();
  // Inject context into the gRPC request metadata
  private final TextMapSetter<Metadata> setter =
      (carrier, key, value) ->
          carrier.put(Metadata.Key.of(key, Metadata.ASCII_STRING_MARSHALLER), value);

  /** Construct client connecting to Ad Service at {@code host:port}. */
  public AdServiceClient(String host, int port) {
    this.serverHostname = host;
    this.serverPort = port;
    this.channel = ManagedChannelBuilder.forAddress(host, port)
                       .usePlaintext()
                       .intercept(new OpenTelemetryClientInterceptor())
                       .build();
    blockingStub = hipstershop.AdServiceGrpc.newBlockingStub(channel);
    // this(
    //     ManagedChannelBuilder.forAddress(host, port)
    //         // Channels are secure by default (via SSL/TLS). For the example we disable TLS to avoid
    //         // needing certificates.
    //         .usePlaintext()
    //         .intercept(new OpenTelemetryClientInterceptor())
    //         .build());
  }

  /** Construct client for accessing RouteGuide server using the existing channel. */
  // private AdServiceClient(ManagedChannel channel) {
  //   this.channel = channel;
  //   blockingStub = hipstershop.AdServiceGrpc.newBlockingStub(channel);
  // }

  public void shutdown() throws InterruptedException {
    channel.shutdown().awaitTermination(5, TimeUnit.SECONDS);
  }

  /** Get Ads from Server. */
  public void getAds(String contextKey) {
    logger.info("Get Ads with context " + contextKey + " ...");
    AdRequest request = AdRequest.newBuilder().addContextKeys(contextKey).build();
    AdResponse response;

    // Start a span
    Span span =
        tracer.spanBuilder("hipstershop.AdService/GetAds").setSpanKind(SpanKind.CLIENT).startSpan();
    span.setAttribute("component", "grpc");
    span.setAttribute("rpc.service", "AdService");
    span.setAttribute("net.peer.ip", this.serverHostname);
    span.setAttribute("net.peer.port", this.serverPort);
    span.setAttribute("pod_name", podName);
    span.setAttribute("node_name", nodeName);
  
    SpanContext spancontext = span.getSpanContext();
    String TraceId = spancontext.getTraceId();
    String SpanId = spancontext.getSpanId();

    try (Scope scope = span.makeCurrent()) {
      logger.info("TraceID:" + TraceId + " SpanID:" + SpanId + " Getting Ads");
      response = blockingStub.getAds(request);
      logger.info("TraceID:" + TraceId + " SpanID:" + SpanId + " Received response from Ads Service.");
    } catch (StatusRuntimeException e) {
      span.setStatus(StatusCode.ERROR, "gRPC status: " + e.getStatus());
      logger.log(Level.WARN, "TraceID:" + TraceId + " SpanID:" + SpanId + " RPC failed: " + e.getStatus());
      return;
    } finally {
      span.end();
    }
    for (Ad ads : response.getAdsList()) {
      logger.info("TraceID:" + TraceId + " SpanID:" + SpanId + " Ads: " + ads.getText());
    }
  }

  public final class OpenTelemetryClientInterceptor implements ClientInterceptor {

    @Override
    public <ReqT, RespT> ClientCall<ReqT, RespT> interceptCall(
        MethodDescriptor<ReqT, RespT> methodDescriptor, CallOptions callOptions, Channel channel) {
      return new ForwardingClientCall.SimpleForwardingClientCall<ReqT, RespT>(
          channel.newCall(methodDescriptor, callOptions)) {
        @Override
        public void start(Listener<RespT> responseListener, Metadata headers) {
          // Inject the request with the current context
          textFormat.inject(Context.current(), headers, setter);
          // Perform the gRPC request
          super.start(responseListener, headers);
        }
      };
    }
  }

  private static int getPortOrDefaultFromArgs(String[] args) {
    int portNumber = 9555;
    if (2 < args.length) {
      try {
        portNumber = Integer.parseInt(args[2]);
      } catch (NumberFormatException e) {
        logger.warn(String.format("Port %s is invalid, use default port %d.", args[2], 9555));
      }
    }
    return portNumber;
  }

  private static String getStringOrDefaultFromArgs(
    String[] args, int index, @Nullable String defaultString) {
    String s = defaultString;
    if (index < args.length) {
      s = args[index];
    }
    return s;
  }

  /**
   * Ads Service Client main. If provided, the first element of {@code args} is the context key to
   * get the ads from the Ads Service
   */
  public static void main(String[] args) throws InterruptedException {
    // Add final keyword to pass checkStyle.
    final String contextKeys = getStringOrDefaultFromArgs(args, 0, "camera");
    final String host = getStringOrDefaultFromArgs(args, 1, "localhost");
    final int serverPort = getPortOrDefaultFromArgs(args);

    // Registers all RPC views.
    // RpcViews.registerAllGrpcViews();

    AdServiceClient client = new AdServiceClient(host, serverPort);
    try {
      client.getAds(contextKeys);
    } finally {
      client.shutdown();
    }

    logger.info("Exiting AdServiceClient...");
  }
}
