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

func RedisErrorString(description string, err error) (string, int) {
	if errors.Is(err, redis.Nil) {
		return "No such key in " + description + " database", http.StatusBadRequest
	} else if err != nil {
		return "Failed to get from " + description + " database", http.StatusInternalServerError
	} else {
		return "Unknown error while working with " + description + " database", http.StatusBadGateway
	}
}

func AnswerRedisError(w http.ResponseWriter, description string, err error) error {
	errorString, code := RedisErrorString(description, err)
	ErrorAsJson(w, errorString, code)
	return err
}

func ErrorAsJson(w http.ResponseWriter, errorString string, code int) {
	errorMap := make(map[string]string)
	errorMap["error"] = errorString
	errorJson, _ := json.Marshal(errorMap)
	http.Error(w, string(errorJson), code)
}
