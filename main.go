// +build !solution

package main

import (
	"encoding/json"
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
	client *redis.Client
)

func main() {
	client = redis.NewClient(&redis.Options{
		Addr:            "db:6379",
		Password:        "", // no password set
		DB:              0,  // use default DB
		MaxRetries:      3,
		MaxRetryBackoff: 5 * time.Second,
	})

	if client == nil {
		log.Fatal("Failed to create Redis Client")
	}

	// Create Stock, Get All Stocks
	http.HandleFunc("/stocks", stocksHandler)

	// Get Stock, Modify Stock, Delete Stock
	http.HandleFunc("/stocks/", stockIdHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func stocksHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		CreateStock(w, r)
	case "", "GET":
		GetAllStocks(w, r)
	default:
		http.Error(w, "Unknown method", http.StatusBadRequest)
	}
}

func CreateStock(w http.ResponseWriter, r *http.Request) {
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

	_, err = client.Set(strconv.FormatUint(stock.Code, 10), contents, 0).Result()
	if err != nil {
		http.Error(w, "Failed to update database", http.StatusInternalServerError)
	}

	_, _ = w.Write(contents)
}

func GetAllStocks(w http.ResponseWriter, r *http.Request) {
	keys, err := client.Keys("*").Result()
	if err != nil {
		http.Error(w, "Failed to get keys from database", http.StatusInternalServerError)
	}

	stocks := make([]Stock, len(keys))
	for i, key := range keys {
		contents, err := client.Get(key).Result()
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
		ModifyStock(w, r, code)
	case "", "GET":
		GetStock(w, r, code)
	case "DELETE":
		DeleteStock(w, r, code)
	default:
		http.Error(w, "Unknown method", http.StatusBadRequest)
	}
}

func ModifyStock(w http.ResponseWriter, r *http.Request, code uint64) {
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

	_, err = client.Set(strconv.FormatUint(stock.Code, 10), contents, 0).Result()
	if err != nil {
		http.Error(w, "Failed to update database", http.StatusInternalServerError)
	}

	_, _ = w.Write(contents)
}

func GetStock(w http.ResponseWriter, r *http.Request, code uint64) {
	stock, err := client.Get(strconv.FormatUint(code, 10)).Result()
	if err != nil {
		http.Error(w, "Failed to get from database", http.StatusInternalServerError)
	}
	if stock == "" {
		http.Error(w, "No such stock", http.StatusNotFound)
	}

	_, _ = w.Write([]byte(stock))
}

func DeleteStock(w http.ResponseWriter, r *http.Request, code uint64) {
	stock, err := client.Get(strconv.FormatUint(code, 10)).Result()
	if err != nil {
		http.Error(w, "Failed to get from database", http.StatusInternalServerError)
	}
	if stock == "" {
		http.Error(w, "No such stock", http.StatusNotFound)
	}

	_, err = client.Del(strconv.FormatUint(code, 10)).Result()
	if err != nil {
		http.Error(w, "Failed to delete from database", http.StatusInternalServerError)
	}
}
