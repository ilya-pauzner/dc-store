package util

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"github.com/go-redis/redis/v7"
	"math"
	"math/big"
	"net/http"
)

func RandomUint64() uint64 {
	bigNumber := math.Pow10(18)
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(bigNumber)))
	return n.Uint64()
}

func AnswerRedisError(w http.ResponseWriter, description string, err error) error {
	if errors.Is(err, redis.Nil) {
		ErrorAsJson(w, "No such key in "+description+" database", http.StatusBadRequest)
	} else if err != nil {
		ErrorAsJson(w, "Failed to get from "+description+" database", http.StatusInternalServerError)
	}
	return err
}

func ErrorAsJson(w http.ResponseWriter, errorString string, code int) {
	errorMap := make(map[string]string)
	errorMap["error"] = errorString
	errorJson, _ := json.Marshal(errorMap)
	http.Error(w, string(errorJson), code)
}
