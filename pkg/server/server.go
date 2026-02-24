package server

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/robbymilo/rgallery/pkg/types"
)

type FilterParams = types.FilterParams
type ResponseAdmin = types.ResponseAdmin
type ResponseProfile = types.ResponseProfile

func DecodeURL(s string) (string, error) {
	decoded, err := url.QueryUnescape(s)
	if err != nil {
		return "", fmt.Errorf("error decoding string: %v", err)
	}
	return decoded, nil
}

func GetHash(hash string) uint32 {
	h, err := strconv.ParseUint(hash, 10, 32)
	if err != nil {
		fmt.Println("error parsing hash:", err)
	}

	return uint32(h)
}
