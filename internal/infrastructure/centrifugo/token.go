package centrifugo

import (
	"fmt"
	"time"

	"go-api/internal/infrastructure/config"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type ConnectionInfo struct {
	Token   string `json:"token"`
	Channel string `json:"channel"`
	WSURL   string `json:"wsUrl"`
}

func NewConnectionInfo(env *config.Config, userID uuid.UUID) (ConnectionInfo, error) {
	channel := UserChannel(userID)
	token, err := generateConnectionToken(env.CentrifugoTokenSecret, userID)
	if err != nil {
		return ConnectionInfo{}, err
	}

	return ConnectionInfo{
		Token:   token,
		Channel: channel,
		WSURL:   env.CentrifugoPublicWSURL,
	}, nil
}

func generateConnectionToken(secret string, userID uuid.UUID) (string, error) {
	// No channels claim: clients may subscribe to namespaces with
	// allow_subscribe_for_client (users + media).
	claims := jwt.MapClaims{
		"sub": userID.String(),
		"exp": time.Now().Add(24 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign centrifugo token: %w", err)
	}

	return signed, nil
}
