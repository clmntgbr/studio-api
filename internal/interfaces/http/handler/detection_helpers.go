package handler

import "strconv"

func parseInt64Query(raw string) (int64, error) {
	return strconv.ParseInt(raw, 10, 64)
}
