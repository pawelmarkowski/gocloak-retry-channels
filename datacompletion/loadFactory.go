package datacompletion

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/pawelmarkowski/gocloak-retry-channels/keycloak"
)

func GetData(ctx context.Context, kc *keycloak.TokenJWT, sourceStr string, wg *sync.WaitGroup, ch chan [][]string) {
	if strings.Contains(sourceStr, "$filter") {
		newOdata(ctx, kc, sourceStr, wg, ch)
		return
	}
	fmt.Errorf("Unsupported Source type passed '%s'", sourceStr)
}
