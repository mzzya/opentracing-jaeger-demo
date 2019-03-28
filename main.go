package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-lib/metrics"

	"github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	jaegerlog "github.com/uber/jaeger-client-go/log"
)

var httpClient *http.Client

func InitTracing() {
	cfg := jaegercfg.Configuration{
		ServiceName: "demo_http_app",
		Sampler: &jaegercfg.SamplerConfig{
			Type:  jaeger.SamplerTypeConst,
			Param: 1,
		},
		Reporter: &jaegercfg.ReporterConfig{
			LogSpans: true,
		},
	}

	// Example logger and metrics factory. Use github.com/uber/jaeger-client-go/log
	// and github.com/uber/jaeger-lib/metrics respectively to bind to real logging and metrics
	// frameworks.
	jLogger := jaegerlog.StdLogger
	jMetricsFactory := metrics.NullFactory

	// Initialize tracer with a logger and a metrics factory
	tracer, _, err := cfg.NewTracer(
		jaegercfg.Logger(jLogger),
		jaegercfg.Metrics(jMetricsFactory),
	)
	if err != nil {
		panic(err)
	}
	// Set the singleton opentracing.Tracer with the Jaeger tracer.
	opentracing.SetGlobalTracer(tracer)
}
func main() {
	InitTracing()
	proxy, err := url.Parse("http://127.0.0.1:8899")
	port := 8080
	if err != nil {
		log.Fatal(err)
	}
	netTransport := &http.Transport{
		Proxy:                 http.ProxyURL(proxy),
		MaxIdleConnsPerHost:   10,
		ResponseHeaderTimeout: time.Second * time.Duration(5),
	}
	httpClient = &http.Client{
		Timeout:   time.Second * 10,
		Transport: netTransport,
	}

	addr := fmt.Sprintf(":%d", port)
	mux := http.NewServeMux()
	mux.HandleFunc("/", indexHandler)
	mux.HandleFunc("/home", homeHandler)
	mux.HandleFunc("/async", serviceHandler)
	mux.HandleFunc("/service", serviceHandler)
	mux.HandleFunc("/db", dbHandler)
	fmt.Printf("Go to http://localhost:%d/home to start a request!\n", port)
	log.Fatal(http.ListenAndServe(addr, mux))
}
func indexHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(`<a href="/home"> Click here to start a request </a>`))
}
func homeHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Request started"))
	sp := opentracing.StartSpan("GET /home") // Start a new root span.
	defer sp.Finish()

	asyncReq, _ := http.NewRequest("GET", "http://localhost:8080/async", nil)
	// Inject the trace information into the HTTP Headers.
	err := sp.Tracer().Inject(sp.Context(), opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(asyncReq.Header))
	if err != nil {
		log.Fatalf("%s: Couldn't inject headers (%v)", r.URL.Path, err)
	}

	go func() {
		sleepMilli(50)
		if _, err := httpClient.Do(asyncReq); err != nil {
			log.Printf("%s: Async call failed (%v)", r.URL.Path, err)
		}
	}()

	sleepMilli(10)
	syncReq, _ := http.NewRequest("GET", "http://localhost:8080/service", nil)
	// Inject the trace info into the headers.
	err = sp.Tracer().Inject(sp.Context(),
		opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(syncReq.Header))
	if err != nil {
		log.Fatalf("%s: Couldn't inject headers (%v)", r.URL.Path, err)
	}
	if _, err = httpClient.Do(syncReq); err != nil {
		log.Printf("%s: Synchronous call failed (%v)", r.URL.Path, err)
		return
	}
	w.Write([]byte("... done!"))
}

func serviceHandler(w http.ResponseWriter, r *http.Request) {
	opName := fmt.Sprintf("%s %s", r.Method, r.URL.Path)
	var sp opentracing.Span
	spCtx, err := opentracing.GlobalTracer().Extract(opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(r.Header))
	if err == nil {
		var sf opentracing.SpanReference
		if r.URL.Path == "/async" {
			sf = opentracing.FollowsFrom(spCtx)

		} else {
			sf = opentracing.ChildOf(spCtx)
		}
		sp = opentracing.StartSpan(opName, sf)
	} else {
		sp = opentracing.StartSpan(opName)
	}
	defer sp.Finish()

	sleepMilli(50)

	dbReq, _ := http.NewRequest("GET", "http://localhost:8080/db", nil)
	err = sp.Tracer().Inject(sp.Context(),
		opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(dbReq.Header))
	if err != nil {
		log.Fatalf("%s: Couldn't inject headers (%v)", r.URL.Path, err)
	}

	if _, err := httpClient.Do(dbReq); err != nil {
		sp.LogEventWithPayload("db request error", err)
	}
}

func dbHandler(w http.ResponseWriter, r *http.Request) {
	var sp opentracing.Span

	spanCtx, err := opentracing.GlobalTracer().Extract(opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(r.Header))
	if err != nil {
		log.Printf("%s: Could not join trace (%v)", r.URL.Path, err)
		return
	}
	if err == nil {
		sp = opentracing.StartSpan("GET /db", opentracing.ChildOf(spanCtx))
	} else {
		sp = opentracing.StartSpan("GET /db")
	}
	defer sp.Finish()
	sleepMilli(25)
}

func sleepMilli(min int) {
	time.Sleep(time.Millisecond * time.Duration(min+rand.Intn(100)))
}
