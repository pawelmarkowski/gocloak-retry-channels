# go-http-poller

## Run app

Export envs
```bash
export CLIENT_ID=xxx
export USERNAME=myemail@example.com
export PASSWORD=strongPassword
export AUTH_URL=https://keycloak.url
export REALM=master
```

```bash
go mod tidy

go run main.go consumer.go producer.go
# or build
go build -o .app
```

## Keycloak module

```mermaid
classDiagram

TokenJWT "1" *-- "0..1" JWT: Contains

class TokenJWT{
    <<struct>>
    +chan[int] RenewRequest
    -timeTime lastRenewRequest
    -gocloakGoCloak client
    -contextContext ctx
    -syncMutex mu
    -client_id string
    -realm string
    -username string
    -password string

    +New(ctx context.Context, auth_url string, client_id string, realm string, username string, password string) (*TokenJWT, error)
    -login() error
    -refresh() error
    +GetToken() *JWT
    -getRenewTime() timeDuration
    +RenewToken(wg *sync.WaitGroup)
    -renewTokenWithRetry() error
}
```

example usage
```go
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
```


## Data completeness validator

Conception

```mermaid
classDiagram

class OdataInput{
    Load()
}

class CSVInput{
    Load()
}

class PsqlInput{
    Load()
}

class LoadInputData{
    <<interface>>
    Load()
}

class DataCompletenessValidator{
    
    
}


class OutputFactory {
    
}

class PsqlStore{
    <<interface>>
}

class FileStore{
    <<interface>>
}
DataCompletenessValidator ..> LoadInputData
LoadInputData ..> OdataInput
LoadInputData ..> CSVInput
LoadInputData ..> PsqlInput
DataCompletenessValidator ..> OutputFactory
OutputFactory ..> PsqlStore
OutputFactory ..> FileStore
```

### Output

Continuous monitoring
```mermaid
graph TD
    A[Start] --> B{"if object <IDENTIFIER> is assigned to sourceA?"};
    B -- Yes --> C[Done];
    B -- No ----> D{if source == A};
    D -- Yes --> E[Assign sourceA to <IDENTIFIER>];
    E --> F[Mark <IDENTIFIER> as assigned];
    F --> C;
    D -- No --> G[Assign sourceB to <IDENTIFIER>];
    G --> C;
```
Simple solution
* set(SourceA) - set(SourceA)

## Bibliography

Helpful websites used during the work on this project

* <https://github.com/golang-standards/project-layout>
* <https://golang.org/doc/effective_go#names>
* mock http request <https://www.thegreatcodeadventure.com/mocking-http-requests-in-golang/>
* graceful shutdown golang <https://callistaenterprise.se/blogg/teknik/2019/10/05/go-worker-cancellation/>
* perfect tool for generating struct for json parsing https://mholt.github.io/json-to-go/
