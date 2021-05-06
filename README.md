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

    +New() (*TokenJWT, error)
    -login() error
    -refresh() error
    +GetToken() *JWT
    -getRenewTime() timeDuration
    +RenewToken()
    -renewTokenWithRetry() error
}
```

## Data completness validator

Conception

```mermaid
classDiagram

class OdataInput{
    <<interface>>
}

class CSVInput{
    <<interface>>
}

class PsqlInput{
    <<interface>>
}

class InputFactory{
    
}

class DataCompletnessValidator{
    
}


class OutputFactory {
    
}

class PsqlStore{
    <<interface>>
}

class FileStore{
    <<interface>>
}
DataCompletnessValidator ..> InputFactory
InputFactory ..> OdataInput
InputFactory ..> CSVInput
InputFactory ..> PsqlInput
DataCompletnessValidator ..> OutputFactory
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
