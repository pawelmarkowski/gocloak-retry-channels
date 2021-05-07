package keycloak

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/Nerzal/gocloak/v8"
	"github.com/avast/retry-go"
)

type TokenJWT struct {
	token           *gocloak.JWT
	RenewRequest    chan int
	lastRenewReqest time.Time
	client          gocloak.GoCloak
	ctx             context.Context
	mu              sync.Mutex
	client_id       string
	realm           string
	username        string
	password        string
}

func New(ctx context.Context, auth_url string, client_id string, realm string, username string, password string) (*TokenJWT, error) {
	token := &TokenJWT{
		RenewRequest:    make(chan int),
		client:          gocloak.NewClient(auth_url),
		ctx:             ctx,
		lastRenewReqest: time.Now(),
		client_id:       client_id,
		realm:           realm,
		username:        username,
		password:        password,
	}
	err := token.login()
	return token, err
}

func (t *TokenJWT) login() error {
	token, err := t.client.Login(
		t.ctx,
		t.client_id,
		"",
		t.realm,
		t.username,
		t.password)
	if err != nil {
		return err
	}
	t.token = token
	return nil
}

func (t *TokenJWT) refresh() error {
	token, err := t.client.RefreshToken(
		t.ctx,
		t.token.RefreshToken,
		os.Getenv("CLIENT_ID"),
		"",
		os.Getenv("REALM"))
	if err != nil {
		return err
	}
	t.token = token
	return nil
}

func (t TokenJWT) GetToken() gocloak.JWT {
	return *t.token
}

func (t TokenJWT) getRenewTime() time.Duration {
	if t.token.ExpiresIn > t.token.RefreshExpiresIn {
		return time.Duration(float64(t.token.RefreshExpiresIn)*0.95) * time.Second
	}
	return time.Duration(float64(t.token.ExpiresIn)*0.95) * time.Second
}

func (t *TokenJWT) RenewToken(wg *sync.WaitGroup) {
	fmt.Println(t.token)
	timer := time.NewTimer(t.getRenewTime())
	defer timer.Stop()
	defer wg.Done()
	for {
		select {
		case <-t.RenewRequest:
			renewTokenWithRetry(t, timer, true)
		case <-timer.C:
			renewTokenWithRetry(t, timer, false)
		case <-t.ctx.Done():
			fmt.Println("RenewToken received cancellation signal, closing RenewRequest!")
			close(t.RenewRequest)
			fmt.Println("RenewToken closed RenewRequest")
			return
		}
	}
}

func renewTokenWithRetry(t *TokenJWT, timer *time.Timer, onRequest bool) error {
	if onRequest == true {
		nextRenew := 30*time.Second - time.Now().Sub(t.lastRenewReqest)
		if nextRenew > 0 {
			fmt.Printf("Renew token on demand skipped (next ondemand renew in %d)\n", nextRenew)
			return nil
		}
	}
	t.mu.Lock()
	err := retry.Do(t.refresh)
	if err != nil {
		fmt.Println("Cannot renew the token", err)
		fmt.Println("Trying to create new token")
		err := retry.Do(t.login)
		if err != nil {
			fmt.Println("Token renewal impossible")
			return err
		}
	}
	t.lastRenewReqest = time.Now()
	if onRequest {
		timer.Stop()
	}
	*timer = *time.NewTimer(t.getRenewTime())
	// uncloak here due to https://github.com/golang/go/issues/40722
	t.mu.Unlock()
	fmt.Println("Token renewal success")
	return nil
}
