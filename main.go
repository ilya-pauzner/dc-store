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

type Stock struct {
	Name       string   `json:"name"`
	Code       uint64   `json:"code,omitempty"`
	Categories []string `json:"categories"`
}

var (
	stocksClient        *redis.Client
	passwordsClient     *redis.Client
	accessTokensClient  *redis.Client
	refreshTokensClient *redis.Client
)

func main() {
	stocksClient = redis.NewClient(&redis.Options{
		Addr: "db:6379",
		DB:   0, // use default DB
	})

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

	// Will be moved to another service
	r.HandleFunc("/register", register).Methods("POST")
	r.HandleFunc("/authorize", authorize).Methods("POST")
	r.HandleFunc("/refresh", refresh).Methods("POST")
	r.HandleFunc("/validate", validate).Methods("POST")

	// createStock, getAllStocks
	r.HandleFunc("/stocks", getAllStocks).Methods("GET")
	r.HandleFunc("/stocks", createStock).Methods("POST")

	// getStock, modifyStock, deleteStock
	r.HandleFunc("/stocks/{code:[0-9]+}", getStock).Methods("GET")
	r.HandleFunc("/stocks/{code:[0-9]+}", modifyStock).Methods("PUT")
	r.HandleFunc("/stocks/{code:[0-9]+}", deleteStock).Methods("DELETE")

	log.Fatal(http.ListenAndServe(":8080", r))
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

	refreshTokenString, ok := data["refresh_token"]
	if !ok {
		http.Error(w, "Failed to get refresh_token from request body", http.StatusBadRequest)
		return
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

func createStock(w http.ResponseWriter, r *http.Request) {
	contents, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	var stock Stock
	err = json.Unmarshal(contents, &stock)
	if err != nil {
		http.Error(w, "Failed to unmarshal request body", http.StatusBadRequest)
		return
	}

	stock.Code = rand.Uint64()

	contents, err = json.Marshal(stock)
	if err != nil {
		http.Error(w, "Failed to marshal response body", http.StatusInternalServerError)
		return
	}

	_, err = stocksClient.Set(strconv.FormatUint(stock.Code, 10), contents, 0).Result()
	if err != nil {
		http.Error(w, "Failed to update database", http.StatusInternalServerError)
		return
	}

	_, _ = w.Write(contents)
}

func getAllStocks(w http.ResponseWriter, _ *http.Request) {
	keys, err := stocksClient.Keys("*").Result()
	if err != nil {
		http.Error(w, "Failed to get keys from database", http.StatusInternalServerError)
		return
	}

	stocks := make([]Stock, len(keys))
	for i, key := range keys {
		contents, err := stocksClient.Get(key).Result()
		if errors.Is(err, redis.Nil) {
			http.Error(w, "No such key in database", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, "Failed to get from database", http.StatusInternalServerError)
			return
		}

		var stock Stock
		er := json.Unmarshal([]byte(contents), &stock)
		if er != nil {
			http.Error(w, "Failed to unmarshal data from database", http.StatusInternalServerError)
			return
		}

		stocks[i] = stock
	}

	contents, er := json.Marshal(stocks)
	if er != nil {
		http.Error(w, "Failed to marshal response body", http.StatusInternalServerError)
		return
	}

	_, _ = w.Write(contents)
}

func modifyStock(w http.ResponseWriter, r *http.Request) {
	// because of regex in router, key exists in vars
	vars := mux.Vars(r)
	codeString := vars["code"]
	code, err := strconv.ParseUint(codeString, 10, 64)
	if err != nil {
		http.Error(w, "Bad stock number", http.StatusBadRequest)
		return
	}

	contents, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	var stock Stock
	err = json.Unmarshal(contents, &stock)
	if err != nil {
		http.Error(w, "Failed to unmarshal request body", http.StatusBadRequest)
		return
	}

	stock.Code = code

	contents, err = json.Marshal(stock)
	if err != nil {
		http.Error(w, "Failed to marshal response body", http.StatusInternalServerError)
		return
	}

	_, err = stocksClient.Set(strconv.FormatUint(stock.Code, 10), contents, 0).Result()
	if err != nil {
		http.Error(w, "Failed to update database", http.StatusInternalServerError)
		return
	}

	_, _ = w.Write(contents)
}

func getStock(w http.ResponseWriter, r *http.Request) {
	// because of regex in router, key exists in vars
	vars := mux.Vars(r)
	codeString := vars["code"]

	stock, err := stocksClient.Get(codeString).Result()
	if errors.Is(err, redis.Nil) {
		http.Error(w, "Wrong stock code", http.StatusForbidden)
		return
	}
	if err != nil {
		http.Error(w, "Failed to get from database", http.StatusInternalServerError)
		return
	}

	_, _ = w.Write([]byte(stock))
}

func deleteStock(w http.ResponseWriter, r *http.Request) {
	// because of regex in router, key exists in vars
	vars := mux.Vars(r)
	codeString := vars["code"]

	_, err := stocksClient.Get(codeString).Result()
	if errors.Is(err, redis.Nil) {
		http.Error(w, "Wrong stock_code", http.StatusForbidden)
		return
	}
	if err != nil {
		http.Error(w, "Failed to get from database", http.StatusInternalServerError)
		return
	}

	_, err = stocksClient.Del(codeString).Result()
	if err != nil {
		http.Error(w, "Failed to delete from database", http.StatusInternalServerError)
		return
	}
}
