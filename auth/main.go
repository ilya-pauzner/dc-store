// +build !solution

package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"github.com/go-redis/redis/v7"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

var (
	passwordsClient     *redis.Client
	accessTokensClient  *redis.Client
	refreshTokensClient *redis.Client
)

func main() {
	passwordsClient = redis.NewClient(&redis.Options{
		Addr: "db:6379",
		DB:   1,
	})

	accessTokensClient = redis.NewClient(&redis.Options{
		Addr: "db:6379",
		DB:   2,
	})

	refreshTokensClient = redis.NewClient(&redis.Options{
		Addr: "db:6379",
		DB:   3,
	})

	r := mux.NewRouter()

	r.HandleFunc("/register", register).Methods("POST")
	r.HandleFunc("/authorize", authorize).Methods("POST")
	r.HandleFunc("/refresh", refresh).Methods("POST")
	r.HandleFunc("/validate", validate).Methods("POST")

	log.Fatal(http.ListenAndServe(":8081", r))
}

func register(w http.ResponseWriter, r *http.Request) {
	contents, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	var data map[string]string
	err = json.Unmarshal(contents, &data)
	if err != nil {
		http.Error(w, "Failed to unmarshal request body", http.StatusBadRequest)
		return
	}

	login, ok := data["login"]
	if !ok {
		http.Error(w, "Failed to get login from request body", http.StatusBadRequest)
		return
	}

	_, err = passwordsClient.Get(login).Result()
	if err == nil {
		http.Error(w, "Login already exists", http.StatusBadRequest)
		return
	} else if !errors.Is(err, redis.Nil) {
		http.Error(w, "Failed to get from database", http.StatusInternalServerError)
		return
	}

	password, ok := data["password"]
	if !ok {
		http.Error(w, "Failed to get password from request body", http.StatusBadRequest)
		return
	}

	hash := sha256.New()
	hashedPassword := hash.Sum([]byte(password))

	_, err = passwordsClient.Set(login, hashedPassword, 0).Result()
	if err != nil {
		http.Error(w, "Failed to update database", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func authorize(w http.ResponseWriter, r *http.Request) {
	contents, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	var data map[string]string
	err = json.Unmarshal(contents, &data)
	if err != nil {
		http.Error(w, "Failed to unmarshal request body", http.StatusBadRequest)
		return
	}

	login, ok := data["login"]
	if !ok {
		http.Error(w, "Failed to get login from request body", http.StatusBadRequest)
		return
	}

	password, ok := data["password"]
	if !ok {
		http.Error(w, "Failed to get password from request body", http.StatusBadRequest)
		return
	}

	hash := sha256.New()
	hashedPassword := hash.Sum([]byte(password))

	hashedPasswordInDataBase, err := passwordsClient.Get(login).Result()
	if !bytes.Equal(hashedPassword, []byte(hashedPasswordInDataBase)) {
		http.Error(w, "Wrong password", http.StatusForbidden)
		return
	}

	tokens := make(map[string]string)

	refreshToken := rand.Uint64()
	refreshTokenString := strconv.FormatUint(refreshToken, 10)
	tokens["refresh_token"] = refreshTokenString
	_, err = refreshTokensClient.Set(refreshTokenString, login, 0).Result()
	if err != nil {
		http.Error(w, "Failed to update database", http.StatusInternalServerError)
		return
	}

	accessToken := rand.Uint64()
	accessTokenString := strconv.FormatUint(accessToken, 10)
	tokens["access_token"] = accessTokenString
	_, err = accessTokensClient.Set(accessTokenString, refreshTokenString, time.Hour).Result()
	if err != nil {
		http.Error(w, "Failed to update database", http.StatusInternalServerError)
		return
	}

	contents, err = json.Marshal(tokens)
	if err != nil {
		http.Error(w, "Failed to marshal response body", http.StatusInternalServerError)
		return
	}

	_, _ = w.Write(contents)
}

func refresh(w http.ResponseWriter, r *http.Request) {
	contents, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	var data map[string]string
	err = json.Unmarshal(contents, &data)
	if err != nil {
		http.Error(w, "Failed to unmarshal request body", http.StatusBadRequest)
		return
	}

	accessTokenString, ok := data["access_token"]
	if !ok {
		http.Error(w, "Failed to get access_token from request body", http.StatusBadRequest)
		return
	}

	refreshTokenFromDb, err := accessTokensClient.Get(accessTokenString).Result()
	if errors.Is(err, redis.Nil) {
		http.Error(w, "Wrong access_token", http.StatusForbidden)
		return
	} else if err != nil {
		http.Error(w, "Failed to get from database", http.StatusInternalServerError)
		return
	}

	refreshTokenString, ok := data["refresh_token"]
	if !ok {
		http.Error(w, "Failed to get refresh_token from request body", http.StatusBadRequest)
		return
	}
	if refreshTokenString != refreshTokenFromDb {
		http.Error(w, "access_token and refresh_token do not form a pair", http.StatusForbidden)
	}

	_, err = refreshTokensClient.Get(refreshTokenString).Result()
	if errors.Is(err, redis.Nil) {
		http.Error(w, "Wrong refresh_token", http.StatusForbidden)
		return
	} else if err != nil {
		http.Error(w, "Failed to get from database", http.StatusInternalServerError)
		return
	}

	tokens := make(map[string]string)

	accessToken := rand.Uint64()
	accessTokenString = strconv.FormatUint(accessToken, 10)
	tokens["access_token"] = accessTokenString
	_, err = accessTokensClient.Set(accessTokenString, refreshTokenString, time.Hour).Result()
	if err != nil {
		http.Error(w, "Failed to update database", http.StatusInternalServerError)
		return
	}

	contents, err = json.Marshal(tokens)
	if err != nil {
		http.Error(w, "Failed to marshal response body", http.StatusInternalServerError)
		return
	}

	_, _ = w.Write(contents)
}

func validate(w http.ResponseWriter, r *http.Request) {
	contents, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	var data map[string]string
	err = json.Unmarshal(contents, &data)
	if err != nil {
		http.Error(w, "Failed to unmarshal request body", http.StatusBadRequest)
		return
	}

	accessTokenString, ok := data["access_token"]
	if !ok {
		http.Error(w, "Failed to get access_token from request body", http.StatusBadRequest)
		return
	}

	_, err = accessTokensClient.Get(accessTokenString).Result()
	if errors.Is(err, redis.Nil) {
		http.Error(w, "Wrong access_token", http.StatusForbidden)
		return
	} else if err != nil {
		http.Error(w, "Failed to get from database", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
