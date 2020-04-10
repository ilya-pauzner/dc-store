// +build !solution

package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"github.com/go-redis/redis/v7"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
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
		Addr:            "db:6379",
		Password:        "", // no password set
		DB:              0,  // use default DB
		MaxRetries:      3,
		MaxRetryBackoff: 5 * time.Second,
	})
	if stocksClient == nil {
		log.Fatal("Failed to create Redis Client")
	}

	passwordsClient = redis.NewClient(&redis.Options{
		Addr:            "db:6379",
		Password:        "", // no password set
		DB:              1,
		MaxRetries:      3,
		MaxRetryBackoff: 5 * time.Second,
	})
	if passwordsClient == nil {
		log.Fatal("Failed to create Redis Client")
	}

	accessTokensClient = redis.NewClient(&redis.Options{
		Addr:            "db:6379",
		Password:        "", // no password set
		DB:              2,
		MaxRetries:      3,
		MaxRetryBackoff: 5 * time.Second,
	})
	if accessTokensClient == nil {
		log.Fatal("Failed to create Redis Client")
	}

	refreshTokensClient = redis.NewClient(&redis.Options{
		Addr:            "db:6379",
		Password:        "", // no password set
		DB:              3,
		MaxRetries:      3,
		MaxRetryBackoff: 5 * time.Second,
	})
	if refreshTokensClient == nil {
		log.Fatal("Failed to create Redis Client")
	}

	// Will be moved to another service
	http.HandleFunc("/register", register)
	http.HandleFunc("/authorize", authorize)
	http.HandleFunc("/refresh", refresh)
	http.HandleFunc("/validate", validate)

	// createStock, getAllStocks
	http.HandleFunc("/stocks", stocksHandler)

	// getStock, modifyStock, deleteStock
	http.HandleFunc("/stocks/", stockIdHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func register(w http.ResponseWriter, r *http.Request) {
	contents, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
	}

	var data map[string]string
	err = json.Unmarshal(contents, &data)
	if err != nil {
		http.Error(w, "Failed to unmarshal request body", http.StatusBadRequest)
	}

	login, ok := data["login"]
	if !ok {
		http.Error(w, "Failed to get login from request body", http.StatusBadRequest)
	}

	password, ok := data["password"]
	if !ok {
		http.Error(w, "Failed to get password from request body", http.StatusBadRequest)
	}

	hash := sha256.New()
	hashedPassword := hash.Sum([]byte(password))

	_, err = passwordsClient.Set(login, hashedPassword, 0).Result()
	if err != nil {
		http.Error(w, "Failed to update database", http.StatusInternalServerError)
	}

	_, _ = w.Write([]byte("Registration successful!"))
}

func authorize(w http.ResponseWriter, r *http.Request) {
	contents, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
	}

	var data map[string]string
	err = json.Unmarshal(contents, &data)
	if err != nil {
		http.Error(w, "Failed to unmarshal request body", http.StatusBadRequest)
	}

	login, ok := data["login"]
	if !ok {
		http.Error(w, "Failed to get login from request body", http.StatusBadRequest)
	}

	password, ok := data["password"]
	if !ok {
		http.Error(w, "Failed to get password from request body", http.StatusBadRequest)
	}

	hash := sha256.New()
	hashedPassword := hash.Sum([]byte(password))

	hashedPasswordInDataBase, err := passwordsClient.Get(login).Result()
	if !bytes.Equal(hashedPassword, []byte(hashedPasswordInDataBase)) {
		http.Error(w, "Wrong password", http.StatusForbidden)
	}

	tokens := make(map[string]string)

	refreshToken := rand.Uint64()
	refreshTokenString := strconv.FormatUint(refreshToken, 10)
	tokens["refresh_token"] = refreshTokenString
	_, err = refreshTokensClient.Set(refreshTokenString, login, 0).Result()
	if err != nil {
		http.Error(w, "Failed to update database", http.StatusInternalServerError)
	}

	accessToken := rand.Uint64()
	accessTokenString := strconv.FormatUint(accessToken, 10)
	tokens["access_token"] = accessTokenString
	_, err = accessTokensClient.Set(accessTokenString, refreshTokenString, time.Hour).Result()
	if err != nil {
		http.Error(w, "Failed to update database", http.StatusInternalServerError)
	}

	contents, err = json.Marshal(tokens)
	if err != nil {
		http.Error(w, "Failed to marshal response body", http.StatusInternalServerError)
	}

	_, _ = w.Write(contents)
}

func refresh(w http.ResponseWriter, r *http.Request) {
	contents, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
	}

	var data map[string]string
	err = json.Unmarshal(contents, &data)
	if err != nil {
		http.Error(w, "Failed to unmarshal request body", http.StatusBadRequest)
	}

	refreshTokenString, ok := data["refresh_token"]
	if !ok {
		http.Error(w, "Failed to get refresh_token from request body", http.StatusBadRequest)
	}

	_, err = refreshTokensClient.Get(refreshTokenString).Result()
	if errors.Is(err, redis.Nil) {
		http.Error(w, "Wrong refresh_token", http.StatusForbidden)
	} else if err != nil {
		http.Error(w, "Failed to get from database", http.StatusInternalServerError)
	}

	tokens := make(map[string]string)

	accessToken := rand.Uint64()
	accessTokenString := strconv.FormatUint(accessToken, 10)
	tokens["access_token"] = accessTokenString
	_, err = accessTokensClient.Set(accessTokenString, refreshTokenString, time.Hour).Result()
	if err != nil {
		http.Error(w, "Failed to update database", http.StatusInternalServerError)
	}

	contents, err = json.Marshal(tokens)
	if err != nil {
		http.Error(w, "Failed to marshal response body", http.StatusInternalServerError)
	}

	_, _ = w.Write(contents)
}

func validate(w http.ResponseWriter, r *http.Request) {
	contents, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
	}

	var data map[string]string
	err = json.Unmarshal(contents, &data)
	if err != nil {
		http.Error(w, "Failed to unmarshal request body", http.StatusBadRequest)
	}

	accessTokenString, ok := data["access_token"]
	if !ok {
		http.Error(w, "Failed to get access_token from request body", http.StatusBadRequest)
	}

	_, err = accessTokensClient.Get(accessTokenString).Result()
	if errors.Is(err, redis.Nil) {
		http.Error(w, "Wrong access_token", http.StatusForbidden)
	} else if err != nil {
		http.Error(w, "Failed to get from database", http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusOK)
}

func stocksHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		createStock(w, r)
	case "", "GET":
		getAllStocks(w, r)
	default:
		http.Error(w, "Unknown method", http.StatusBadRequest)
	}
}

func createStock(w http.ResponseWriter, r *http.Request) {
	contents, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
	}

	var stock Stock
	err = json.Unmarshal(contents, &stock)
	if err != nil {
		http.Error(w, "Failed to unmarshal request body", http.StatusBadRequest)
	}

	stock.Code = rand.Uint64()

	contents, err = json.Marshal(stock)
	if err != nil {
		http.Error(w, "Failed to marshal response body", http.StatusInternalServerError)
	}

	_, err = stocksClient.Set(strconv.FormatUint(stock.Code, 10), contents, 0).Result()
	if err != nil {
		http.Error(w, "Failed to update database", http.StatusInternalServerError)
	}

	_, _ = w.Write(contents)
}

func getAllStocks(w http.ResponseWriter, r *http.Request) {
	keys, err := stocksClient.Keys("*").Result()
	if err != nil {
		http.Error(w, "Failed to get keys from database", http.StatusInternalServerError)
	}

	stocks := make([]Stock, len(keys))
	for i, key := range keys {
		contents, err := stocksClient.Get(key).Result()
		if err != nil {
			http.Error(w, "Failed to get from database", http.StatusInternalServerError)
		}
		if contents == "" {
			http.Error(w, "No such stock", http.StatusNotFound)
		}

		var stock Stock
		er := json.Unmarshal([]byte(contents), &stock)
		if er != nil {
			http.Error(w, "Failed to unmarshal data from database", http.StatusInternalServerError)
		}

		stocks[i] = stock
	}

	contents, er := json.Marshal(stocks)
	if er != nil {
		http.Error(w, "Failed to marshal response body", http.StatusInternalServerError)
	}

	_, _ = w.Write(contents)
}

func stockIdHandler(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	key := parts[len(parts)-1]

	code, err := strconv.ParseUint(key, 10, 64)
	if err != nil {
		http.Error(w, "Bad stock number", http.StatusBadRequest)
	}

	switch r.Method {
	case "PUT":
		modifyStock(w, r, code)
	case "", "GET":
		getStock(w, r, code)
	case "DELETE":
		deleteStock(w, r, code)
	default:
		http.Error(w, "Unknown method", http.StatusBadRequest)
	}
}

func modifyStock(w http.ResponseWriter, r *http.Request, code uint64) {
	contents, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
	}

	var stock Stock
	err = json.Unmarshal(contents, &stock)
	if err != nil {
		http.Error(w, "Failed to unmarshal request body", http.StatusBadRequest)
	}

	stock.Code = code

	contents, err = json.Marshal(stock)
	if err != nil {
		http.Error(w, "Failed to marshal response body", http.StatusInternalServerError)
	}

	_, err = stocksClient.Set(strconv.FormatUint(stock.Code, 10), contents, 0).Result()
	if err != nil {
		http.Error(w, "Failed to update database", http.StatusInternalServerError)
	}

	_, _ = w.Write(contents)
}

func getStock(w http.ResponseWriter, r *http.Request, code uint64) {
	stock, err := stocksClient.Get(strconv.FormatUint(code, 10)).Result()
	if err != nil {
		http.Error(w, "Failed to get from database", http.StatusInternalServerError)
	}
	if stock == "" {
		http.Error(w, "No such stock", http.StatusNotFound)
	}

	_, _ = w.Write([]byte(stock))
}

func deleteStock(w http.ResponseWriter, r *http.Request, code uint64) {
	stock, err := stocksClient.Get(strconv.FormatUint(code, 10)).Result()
	if err != nil {
		http.Error(w, "Failed to get from database", http.StatusInternalServerError)
	}
	if stock == "" {
		http.Error(w, "No such stock", http.StatusNotFound)
	}

	_, err = stocksClient.Del(strconv.FormatUint(code, 10)).Result()
	if err != nil {
		http.Error(w, "Failed to delete from database", http.StatusInternalServerError)
	}
}
