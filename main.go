package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/pawelmarkowski/gocloak-retry-channels/keycloak"
)

const workerPoolSize = 2

func main() {
	token, err := keycloak.New()
	if err != nil {
		panic(err)
	}
	go token.RenewToken()
	// create the consumer
	consumer := Consumer{
		ingestChan: make(chan int, 1),
		jobsChan:   make(chan int, workerPoolSize),
	}

	// Simulate external lib sending us 10 events per second
	producer := Producer{callbackFunc: consumer.callbackFunc}
	go producer.start()

	// Set up cancellation context and waitgroup
	ctx, cancelFunc := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}

	// Start consumer with cancellation context passed
	go consumer.startConsumer(ctx)

	// Start workers and Add [workerPoolSize] to WaitGroup
	wg.Add(workerPoolSize)
	for i := 0; i < workerPoolSize; i++ {
		go consumer.workerFunc(wg, i, token.RenewRequest)
	}

	// Handle sigterm and await termChan signal
	termChan := make(chan os.Signal)
	signal.Notify(termChan, syscall.SIGINT, syscall.SIGTERM)

	<-termChan // Blocks here until interrupted

	// Handle shutdown
	fmt.Println("*********************************\nShutdown signal received\n*********************************")
	cancelFunc() // Signal cancellation to context.Context
	wg.Wait()    // Block here until are workers are done

	fmt.Println("All workers done, shutting down!")
}
