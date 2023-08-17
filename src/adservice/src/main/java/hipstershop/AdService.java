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

import com.google.common.collect.ImmutableListMultimap;
import com.google.common.collect.ImmutableMap;
import com.google.common.collect.Iterables;
import hipstershop.Demo.Ad;
import hipstershop.Demo.AdRequest;
import hipstershop.Demo.AdResponse;
import io.grpc.services.*;
import io.grpc.Contexts;
import io.grpc.Grpc;
import io.grpc.Metadata;
import io.grpc.Server;
import io.grpc.Status;
import io.grpc.ServerBuilder;
import io.grpc.ServerCall;
import io.grpc.ServerCallHandler;
import io.grpc.stub.StreamObserver;
import io.grpc.StatusRuntimeException;
import io.grpc.health.v1.HealthCheckResponse.ServingStatus;
import java.io.IOException;
import java.util.ArrayList;
import java.util.Collection;
import java.util.List;
import java.util.Random;
import java.util.HashMap;
import java.util.concurrent.TimeUnit;
import java.net.InetSocketAddress;
import org.apache.logging.log4j.Level;
import org.apache.logging.log4j.LogManager;
import org.apache.logging.log4j.Logger;
import io.opentelemetry.api.OpenTelemetry;
import io.opentelemetry.api.trace.Span;
import io.opentelemetry.api.trace.SpanContext;
import io.opentelemetry.api.trace.StatusCode;
import io.opentelemetry.extension.annotations.WithSpan;
import java.sql.*;
import java.lang.ClassNotFoundException;


public final class AdService {
  // Logger.getGlobal().setLevel(Level.ALL);
  private static final Logger logger = LogManager.getLogger(AdService.class);
  // logger.setLevel(Level.ALL);
  private static final String mysqlAddr = System.getenv("MYSQL_ADDR");
  private static final String user = System.getenv("SQL_USER");
  private static final String pwd = System.getenv("SQL_PASSWORD");
  private static final String podName = System.getenv("POD_NAME");
  private static final String nodeName = System.getenv("NODE_NAME");

  @SuppressWarnings("FieldCanBeLocal")
  private static int MAX_ADS_TO_SERVE = 2;

  private Server server;
  private HealthStatusManager healthMgr;

  private static final AdService service = new AdService();

  private void start() throws IOException {
    int port = Integer.parseInt(System.getenv().getOrDefault("PORT", "9555"));
    healthMgr = new HealthStatusManager();

    server =
        ServerBuilder.forPort(port)
            .addService(new AdServiceImpl())
            .addService(healthMgr.getHealthService())
            .build()
            .start();
    logger.info("Ad Service started, listening on " + port);
    Runtime.getRuntime()
        .addShutdownHook(
            new Thread(
                () -> {
                  // Use stderr here since the logger may have been reset by its JVM shutdown hook.
                  System.err.println(
                      "*** shutting down gRPC ads server since JVM is shutting down");
                  AdService.this.stop();
                  System.err.println("*** server shut down");
                }));
    healthMgr.setStatus("", ServingStatus.SERVING);
  }

  private void stop() {
    if (server != null) {
      healthMgr.clearStatus("");
      server.shutdown();
    }
  }

  private static class AdServiceImpl extends hipstershop.AdServiceGrpc.AdServiceImplBase {

    /**
     * Retrieves ads based on context provided in the request {@code AdRequest}.
     *
     * @param req the request containing context.
     * @param responseObserver the stream observer which gets notified with the value of {@code
     *     AdResponse}
     */
    @Override
    @WithSpan
    public void getAds(AdRequest req, StreamObserver<AdResponse> responseObserver){
      AdService service = AdService.getInstance();

      Span span = span = Span.current();
      span.updateName("hipstershop.AdServiceImpl/getAds");
      span.setAttribute("PodName", podName);
      span.setAttribute("NodeName", nodeName);
      SpanContext spancontext = span.getSpanContext();
      String TraceId = spancontext.getTraceId();
      String SpanId = spancontext.getSpanId();

      try {
        List<Ad> allAds = new ArrayList<>();
        logger.info("TraceID: " + TraceId + " SpanID: " + SpanId + " Received ad request (context_words=" + req.getContextKeysList() + ")");
        if (req.getContextKeysCount() > 0) {
          for (int i = 0; i < req.getContextKeysCount(); i++) {
            Collection<Ad> ads = service.getAdsByCategory(req.getContextKeys(i));
            allAds.addAll(ads);
          }
        }

        /**
          Inject return fault for isEmpty() by Byteman, isEmpty() always return True
        **/
        if (allAds.isEmpty()) {
          // Serve random ads.
          /**
             Inject exception fault and CPU comsumption by Byteman
          **/
          logger.info("TraceID: " + TraceId + " SpanID: " + SpanId + " No context provided. Constructe random Ads start");
          allAds = service.getRandomAds();
          logger.info("TraceID: " + TraceId + " SpanID: " + SpanId + " Constructe random Ads finish");
          // throw new StatusRuntimeException(Status.UNAVAILABLE);
        }

        AdResponse reply = AdResponse.newBuilder().addAllAds(allAds).build();
        responseObserver.onNext(reply);
        responseObserver.onCompleted();
        
      } catch (Exception e) {        
        logger.log(Level.WARN, "TraceID: " + TraceId + " SpanID: " + SpanId + " Get Ads Failed with status");
        // List<Ad> allAds = new ArrayList<>();
        // AdResponse reply = AdResponse.newBuilder().addAllAds(allAds).build();
        // responseObserver.onNext(reply);
        // responseObserver.onCompleted();
        responseObserver.onError(e);
      }
    }
  }
 
  // private static final ImmutableListMultimap<String, Ad> adsMap = createAdsMap();

  @WithSpan
  private Collection<Ad> getAdsByCategory(String category) {
    ImmutableListMultimap<String, Ad> adsMap = createAdsMap();
    Span categorySpan = Span.current();
    categorySpan.updateName("hipstershop.AdService/getAdsByCategory");
    categorySpan.setAttribute("PodName", podName);
    categorySpan.setAttribute("NodeName", nodeName);

    SpanContext spancontext = categorySpan.getSpanContext();
    String TraceId = spancontext.getTraceId();
    String SpanId = spancontext.getSpanId();

    logger.info("TraceID: " + TraceId + " SpanID: " + SpanId + " Get Ads by Category start");
    Collection<Ad> ads= adsMap.get(category);
    logger.info("TraceID: " + TraceId + " SpanID: " + SpanId + " Get Ads by Category finish");

    return ads;
  }

  private static final Random random = new Random();

  @WithSpan
  private List<Ad> getRandomAds() {
    ImmutableListMultimap<String, Ad> adsMap = createAdsMap();

    Span randomSpan = Span.current();
    randomSpan.setAttribute("PodName", podName);
    randomSpan.setAttribute("NodeName", nodeName);
    randomSpan.updateName("hipstershop.AdService/getRandomAds");
    SpanContext spancontext = randomSpan.getSpanContext();
    String TraceId = spancontext.getTraceId();
    String SpanId = spancontext.getSpanId();
  
    List<Ad> ads = new ArrayList<>(MAX_ADS_TO_SERVE);
    Collection<Ad> allAds = adsMap.values();

    try {
      for (int i = 0; i < MAX_ADS_TO_SERVE; i++) {
        ads.add(Iterables.get(allAds, random.nextInt(allAds.size())));
      }
    } catch (StatusRuntimeException e) {
      logger.log(Level.WARN, "TraceID: " + TraceId + " SpanID: " + SpanId + " Constructe random Ads Failed with status {}", e.getStatus());
    }

    return ads;
  }

  private static AdService getInstance() {
    return service;
  }

  /** Await termination on the main thread since the grpc library uses daemon threads. */
  private void blockUntilShutdown() throws InterruptedException {
    if (server != null) {
      server.awaitTermination();
    }
  }

  private static ImmutableListMultimap<String, Ad> createAdsMap() {
    Span adsSpan = Span.current();
    adsSpan.setAttribute("PodName", podName);
    adsSpan.setAttribute("NodeName", nodeName);
    adsSpan.updateName("hipstershop.AdService/createAdsMap");
    SpanContext spancontext = adsSpan.getSpanContext();
    String TraceId = spancontext.getTraceId();
    String SpanId = spancontext.getSpanId();

    HashMap<String, Ad> aditemMap = new HashMap<String, Ad>();
    // connect to mysql
    String JDBC_DRIVER = "com.mysql.cj.jdbc.Driver";
    String DB_URL = "jdbc:mysql://" + mysqlAddr + "/addatabase?useSSL=false&allowPublicKeyRetrieval=true&serverTimezone=UTC";

    // set user and pwd
    String USER = user;
    String PASS = pwd;
        
    Connection conn = null;
    Statement stmt = null;

    try {
      // open link
      Class.forName(JDBC_DRIVER);
      conn = DriverManager.getConnection(DB_URL, USER, PASS);

      // execute query
      stmt = conn.createStatement();
      String sql;
      sql = "SELECT * FROM aditems";
      ResultSet rs = stmt.executeQuery(sql);

      // spread result set
      while (rs.next()) {
        String item_name = rs.getString("item_name");
        String redirecturl = rs.getString("redirect_url");
        String text = rs.getString("text");

        aditemMap.put(item_name, Ad.newBuilder()
                                    .setRedirectUrl(redirecturl)
                                    .setText(text)
                                    .build());
      }
      rs.close();
      stmt.close();
      conn.close();
      logger.info("TraceID: " + TraceId + " SpanID: " + SpanId  + " Query items successfully.");     
    } catch (ClassNotFoundException e) {
      logger.fatal("TraceID: " + TraceId + " SpanID: " + SpanId  + " MYSQL JDBC DRIVER loading error. JDBC_DRIVER: " + JDBC_DRIVER);
      return null;
    } catch(SQLException se) {
      logger.fatal("TraceID: " + TraceId + " SpanID: " + SpanId  + " SQL read error. JDBC_DRIVER: " + JDBC_DRIVER + ". MYSQL DB_URL: " + DB_URL);
      return null;
    } 

    return ImmutableListMultimap.<String, Ad>builder()
        .putAll("photography", aditemMap.get("camera"), aditemMap.get("lens"))
        .putAll("vintage", aditemMap.get("camera"), aditemMap.get("lens"), aditemMap.get("recordPlayer"))
        .put("cycling", aditemMap.get("bike"))
        .put("cookware", aditemMap.get("baristaKit"))
        .putAll("gardening", aditemMap.get("airPlant"), aditemMap.get("terrarium"))
        .build();
  }
  // private static ImmutableListMultimap<String, Ad> createAdsMap() {
  //   Ad camera =
  //       Ad.newBuilder()
  //           .setRedirectUrl("/product/2ZYFJ3GM2N")
  //           .setText("Film camera for sale. 50% off.")
  //           .build();
  //   Ad lens =
  //       Ad.newBuilder()
  //           .setRedirectUrl("/product/66VCHSJNUP")
  //           .setText("Vintage camera lens for sale. 20% off.")
  //           .build();
  //   Ad recordPlayer =
  //       Ad.newBuilder()
  //           .setRedirectUrl("/product/0PUK6V6EV0")
  //           .setText("Vintage record player for sale. 30% off.")
  //           .build();
  //   Ad bike =
  //       Ad.newBuilder()
  //           .setRedirectUrl("/product/9SIQT8TOJO")
  //           .setText("City Bike for sale. 10% off.")
  //           .build();
  //   Ad baristaKit =
  //       Ad.newBuilder()
  //           .setRedirectUrl("/product/1YMWWN1N4O")
  //           .setText("Home Barista kitchen kit for sale. Buy one, get second kit for free")
  //           .build();
  //   Ad airPlant =
  //       Ad.newBuilder()
  //           .setRedirectUrl("/product/6E92ZMYYFZ")
  //           .setText("Air plants for sale. Buy two, get third one for free")
  //           .build();
  //   Ad terrarium =
  //       Ad.newBuilder()
  //           .setRedirectUrl("/product/L9ECAV7KIM")
  //           .setText("Terrarium for sale. Buy one, get second one for free")
  //           .build();
  //   return ImmutableListMultimap.<String, Ad>builder()
  //       .putAll("photography", camera, lens)
  //       .putAll("vintage", camera, lens, recordPlayer)
  //       .put("cycling", bike)
  //       .put("cookware", baristaKit)
  //       .putAll("gardening", airPlant, terrarium)
  //       .build();
  // }

  /** Main launches the server from the command line. */
  public static void main(String[] args) throws IOException, InterruptedException {
    // Start the RPC server. You shouldn't see any output from gRPC before this.
    logger.info("AdService starting.");
    final AdService service = AdService.getInstance();
    service.start();
    service.blockUntilShutdown();
  }
}
