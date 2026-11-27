package centrifugo

import "github.com/google/uuid"

func UserChannel(userID uuid.UUID) string {
	return "users:" + userID.String()
}

func MediaChannel(mediaID uuid.UUID) string {
	return "media:" + mediaID.String()
}
