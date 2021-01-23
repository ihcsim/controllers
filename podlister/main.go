package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func main() {
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Printf("can't get in-cluster config: %s", err)
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Printf("can't set up clientset: %s", err)
		os.Exit(1)
	}

	var contextTimeout time.Duration
	if v, exists := os.LookupEnv("CONTEXT_TIMEOUT"); exists {
		parsed, err := time.ParseDuration(v)
		if err != nil {
			log.Printf("bad env var: %s", err)
			os.Exit(1)
		}
		contextTimeout = parsed
	}

	tickInterval := time.Millisecond * 100
	if v, exists := os.LookupEnv("TICK_INTERVAL"); exists {
		parsed, err := time.ParseDuration(v)
		if err != nil {
			log.Printf("bad env var: %s", err)
			os.Exit(1)
		}
		tickInterval = parsed
	}

	targetNamespace := "demo"
	if v, exists := os.LookupEnv("TARGET_NAMESPACE"); exists {
		targetNamespace = v
	}

	log.Printf("target namespace: %s, tick interval: %s",
		targetNamespace, tickInterval)

	var (
		dataChan            = make(chan string)
		errChan             = make(chan error)
		exit                = make(chan os.Signal, 1)
		mainCtx, mainCancel = context.WithCancel(context.Background())
		listOpts            = metav1.ListOptions{}
	)

	signal.Notify(exit, os.Interrupt, os.Kill)

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
					var (
						ctx    = mainCtx
						cancel = mainCancel
					)

					if contextTimeout > 0 {
						log.Printf("context timeout: %s", contextTimeout)
						ctx, cancel = context.WithTimeout(mainCtx, contextTimeout)
					}
					defer cancel()

					pods, err := clientset.CoreV1().Pods(targetNamespace).List(ctx, listOpts)
					if err != nil {
						log.Printf("error while listing pods: %s", err)
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

type Result struct {
	Namespace string   `json:"namespace"`
	Pods      []string `json:"pods"`
}
