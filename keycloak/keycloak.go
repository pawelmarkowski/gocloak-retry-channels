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
}

func New() (*TokenJWT, error) {
	token := &TokenJWT{
		RenewRequest:    make(chan int),
		client:          gocloak.NewClient(os.Getenv("AUTH_URL")),
		ctx:             context.Background(),
		lastRenewReqest: time.Now()}
	err := token.login()
	return token, err
}

func (t *TokenJWT) login() error {
	token, err := t.client.Login(
		t.ctx,
		os.Getenv("CLIENT_ID"),
		"",
		os.Getenv("REALM"),
		os.Getenv("USERNAME"),
		os.Getenv("PASSWORD"))
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

func (t *TokenJWT) RenewToken() {
	fmt.Println(t.token)
	timer := time.NewTimer(t.getRenewTime())
	defer timer.Stop()
	for {
		select {
		case <-t.RenewRequest:
			renewTokenWithRetry(t, timer, true)
		case <-timer.C:
			renewTokenWithRetry(t, timer, false)
		}
	}
}

func renewTokenWithRetry(t *TokenJWT, timer *time.Timer, onRequest bool) error {
	if onRequest == true {
		nextRenew := 10*time.Second - time.Now().Sub(t.lastRenewReqest)
		if nextRenew > 0 {
			fmt.Printf("Renew token on demand skipped (next ondemand renew in %d)\n", nextRenew)
			return nil
		}
	}
	t.mu.Lock()
	err := retry.Do(func() error { return t.refresh() })
	if err != nil {
		fmt.Println("Cannot renew the token", err)
		fmt.Println("Trying to create new token")
		err := retry.Do(func() error { return t.login() })
		if err != nil {
			fmt.Println("Token renewal impossible")
			return err
		}
	}
	t.lastRenewReqest = time.Now()
	if onRequest == true {
		timer.Stop()
	}
	t.token.ExpiresIn = 5
	*timer = *time.NewTimer(t.getRenewTime())
	// uncloak here due to https://github.com/golang/go/issues/40722
	t.mu.Unlock()
	fmt.Println("Token renewal success")
	return nil
}
