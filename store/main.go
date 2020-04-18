// +build !solution

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/go-redis/redis/v7"
	"github.com/gorilla/mux"
	_ "github.com/streadway/amqp"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"strconv"
)

type Stock struct {
	Name       string   `json:"name"`
	Code       uint64   `json:"code,omitempty"`
	Categories []string `json:"categories"`
}

var (
	stocksClient *redis.Client
)

func main() {
	stocksClient = redis.NewClient(&redis.Options{
		Addr: "db:6379",
		DB:   0, // use default DB
	})

	r := mux.NewRouter()

	// createStock, getAllStocks
	r.HandleFunc("/stocks", getAllStocks).Methods("GET")
	r.HandleFunc("/stocks", createStock).Methods("POST")

	// getStock, modifyStock, deleteStock
	r.HandleFunc("/stocks/{code:[0-9]+}", getStock).Methods("GET")
	r.HandleFunc("/stocks/{code:[0-9]+}", modifyStock).Methods("PUT")
	r.HandleFunc("/stocks/{code:[0-9]+}", deleteStock).Methods("DELETE")

	log.Fatal(http.ListenAndServe(":8080", r))
}

func validate(contents []byte) (bool, error) {
	answer, err := http.Post("http://auth:8081/validate", "application/json", bytes.NewReader(contents))
	if err != nil {
		return false, err
	}
	if answer.StatusCode != http.StatusOK {
		return false, nil
	}
	return true, nil
}

func createStock(w http.ResponseWriter, r *http.Request) {
	contents, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	ok, err := validate(contents)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, "Access denied", http.StatusForbidden)
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

func getAllStocks(w http.ResponseWriter, r *http.Request) {
	contents, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	ok, err := validate(contents)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

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
	contents, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	ok, err := validate(contents)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// because of regex in router, key exists in vars
	vars := mux.Vars(r)
	codeString := vars["code"]
	code, err := strconv.ParseUint(codeString, 10, 64)
	if err != nil {
		http.Error(w, "Bad stock number", http.StatusBadRequest)
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

	_, err = stocksClient.Set(codeString, contents, 0).Result()
	if err != nil {
		http.Error(w, "Failed to update database", http.StatusInternalServerError)
		return
	}

	_, _ = w.Write(contents)
}

func getStock(w http.ResponseWriter, r *http.Request) {
	contents, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	ok, err := validate(contents)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

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
	contents, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	ok, err := validate(contents)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// because of regex in router, key exists in vars
	vars := mux.Vars(r)
	codeString := vars["code"]

	_, err = stocksClient.Get(codeString).Result()
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
