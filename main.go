package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/pawelmarkowski/gocloak-retry-channels/datacompletion"
	"github.com/pawelmarkowski/gocloak-retry-channels/keycloak"
)

func difference(b, a [][]string) [][]string {
	mb := make(map[string]struct{}, len(b))
	for _, x := range b {
		mb[x[0]] = struct{}{}
	}
	var diff [][]string
	for _, x := range a {
		if _, found := mb[x[0]]; !found {
			diff = append(diff, []string{x[0]})
		}
	}
	return diff
}

func main() {
	// Set up cancellation context and waitgroup
	ctx, cancelFunc := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}

	token, err := keycloak.New(
		ctx,
		os.Getenv("AUTH_URL"),
		os.Getenv("CLIENT_ID"),
		os.Getenv("REALM"),
		os.Getenv("USERNAME"),
		os.Getenv("PASSWORD"))
	if err != nil {
		panic(err)
	}
	// // Start keycloak token control and Add [workerPoolSize] to WaitGroup
	wg.Add(1)
	go token.RenewToken(wg)

	channelSourceA := make(chan [][]string)
	wg.Add(1)
	go datacompletion.GetData(ctx, token, os.Getenv("SOURCE_URL"), wg, channelSourceA)
	channelSourceB := make(chan [][]string)
	wg.Add(1)
	go datacompletion.GetData(ctx, token, os.Getenv("SOURCE2_URL"), wg, channelSourceB)
	channelSourceC := make(chan [][]string)
	wg.Add(1)
	go datacompletion.GetData(ctx, token, os.Getenv("SOURCE3_URL"), wg, channelSourceC)
	s1Data := <-channelSourceA
	s2Data := <-channelSourceB
	s3Data := <-channelSourceC
	datacompletion.SaveResults("s1missing.csv", difference(s3Data, s1Data))
	datacompletion.SaveResults("s2missing.csv", difference(s3Data, s2Data))
	// // create the consumer
	// consumer := Consumer{
	// 	ingestChan: make(chan int, 1),
	// 	jobsChan:   make(chan int, workerPoolSize),
	// }

	// // Simulate external lib sending us 10 events per second
	// producer := Producer{callbackFunc: consumer.callbackFunc}
	// go producer.start()

	// // Start consumer with cancellation context passed
	// go consumer.startConsumer(ctx)

	// // Start workers and Add [workerPoolSize] to WaitGroup
	// wg.Add(workerPoolSize)
	// for i := 0; i < workerPoolSize; i++ {
	// 	go consumer.workerFunc(wg, i, token.RenewRequest)
	// }

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
