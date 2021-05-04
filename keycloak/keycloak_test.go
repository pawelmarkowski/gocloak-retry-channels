package keycloak

import (
	"testing"
	"time"

	"github.com/Nerzal/gocloak/v8"
	"github.com/stretchr/testify/assert"
)

func TestTimeToRenew(t *testing.T) {
	token := &TokenJWT{
		token: &gocloak.JWT{
			ExpiresIn:        60 * 20,
			RefreshExpiresIn: 60 * 15},
		lastRenewReqest: time.Now(),
		client:          nil,
		ctx:             nil,
	}
	value := token.getRenewTime()
	assert.Greater(t, time.Duration(token.token.RefreshExpiresIn)*time.Second, value)
}

// TODO
