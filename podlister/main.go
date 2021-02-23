package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func main() {
	var contextTimeout time.Duration
	if v, exists := os.LookupEnv("CONTEXT_TIMEOUT"); exists {
		parsed, err := time.ParseDuration(v)
		if err != nil {
			log.Fatalf("bad env var: %s", err)
		}
		contextTimeout = parsed
	}

	tickInterval := time.Millisecond * 100
	if v, exists := os.LookupEnv("TICK_INTERVAL"); exists {
		parsed, err := time.ParseDuration(v)
		if err != nil {
			log.Fatalf("bad env var: %s", err)
		}
		tickInterval = parsed
	}

	namespace := "demo"
	if v, exists := os.LookupEnv("NAMESPACE"); exists {
		namespace = v
	}

	targetNamespace := "demo"
	if v, exists := os.LookupEnv("TARGET_NAMESPACE"); exists {
		targetNamespace = v
	}

	pod := ""
	if v, exists := os.LookupEnv("POD_NAME"); exists {
		pod = v
	}

	showErrorsOnly := true
	if v, exists := os.LookupEnv("SHOW_ERRORS_ONLY"); exists {
		if parsed, err := strconv.ParseBool(v); err == nil {
			showErrorsOnly = parsed
		}
	}

	log.Printf("namespace: %s, pod: %s, target namespace: %s, tick interval: %s, show errors only: %t",
		namespace, pod, targetNamespace, tickInterval, showErrorsOnly)

	var (
		dataChan            = make(chan string)
		errChan             = make(chan error)
		exit                = make(chan os.Signal, 1)
		mainCtx, mainCancel = context.WithCancel(context.Background())
		listOpts            = metav1.ListOptions{}
	)

	signal.Notify(exit, os.Interrupt, os.Kill)

	clientset, err := clientSet()
	if err != nil {
		log.Fatalf("can't set up K8s clientset: %s", err)
	}

	requestCounter, err := registerCounter()
	if err != nil {
		log.Fatalf("fail to register Prometheus counter: %s", err)
	}

	server, err := startHTTPServer()
	defer func() {
		ctx, cancel := context.WithTimeout(mainCtx, time.Second*5)
		defer cancel()
		server.Shutdown(ctx)
	}()

	go func() {
		for {
			select {
			case <-mainCtx.Done():
				msg := "exiting goroutine"
				if err := mainCtx.Err(); err != nil {
					msg += fmt.Sprintf(": %s", err)
				}
				log.Print(msg)
				break

			case <-time.Tick(tickInterval):
				go func() {
					ctx, cancel := context.WithCancel(mainCtx)
					if contextTimeout > 0 {
						ctx, cancel = context.WithTimeout(mainCtx, contextTimeout)
					}
					defer cancel()

					requestLabels := prometheus.Labels{
						"namespace":        namespace,
						"pod":              pod,
						"target_namespace": targetNamespace,
					}
					requestCounter.With(requestLabels).Inc()

					pods, err := clientset.CoreV1().Pods(targetNamespace).List(ctx, listOpts)
					if err != nil {
						errChan <- fmt.Errorf("error while listing pods: %s", err)
						return
					}

					if showErrorsOnly {
						return
					}

					if len(pods.Items) == 0 {
						dataChan <- fmt.Sprintf("no pods found in namespace %s", targetNamespace)
						return
					}

					foundPods := []string{}
					for _, pod := range pods.Items {
						foundPods = append(foundPods, pod.Name)
					}

					dataChan <- fmt.Sprintf("%s", Result{Namespace: targetNamespace, Pods: foundPods})
				}()
			}
		}
	}()

	for {
		select {
		case err := <-errChan:
			log.Println(err)
		case data := <-dataChan:
			log.Println(data)
		case <-exit:
			mainCancel()
			break
		}
	}
}

func registerCounter() (*prometheus.CounterVec, error) {
	counter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "demo_http_requests_total",
			Help: "A counter for requests issued by the demo controllers.",
		},
		[]string{"namespace", "target_namespace", "pod"},
	)

	if err := prometheus.Register(counter); err != nil {
		return nil, err
	}

	return counter, nil
}

func clientSet() (*kubernetes.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(config)
}

func startHTTPServer() (*http.Server, error) {
	server := &http.Server{Addr: ":8080"}
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		server.Handler = mux

		if err := server.ListenAndServe(); err != nil {
			log.Println(err)
		}
	}()

	return server, nil
}

type Result struct {
	Namespace string   `json:"namespace"`
	Pods      []string `json:"pods"`
}
