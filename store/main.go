// +build !solution

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/go-redis/redis/v7"
	"github.com/gorilla/mux"
	"github.com/ilya-pauzner/dc-store/util"
	"log"
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

func validateAndAnswer(w http.ResponseWriter, header http.Header) bool {
	emptyBuffer := make([]byte, 0)
	emptyReader := bytes.NewReader(emptyBuffer)
	req, err := http.NewRequest(http.MethodPost, "http://auth:8081/validate", emptyReader)
	if err != nil {
		util.ErrorAsJson(w, err.Error(), http.StatusInternalServerError)
		return false
	}

	req.Header = header

	answer, err := http.DefaultClient.Do(req)
	if err != nil {
		util.ErrorAsJson(w, err.Error(), http.StatusInternalServerError)
		return false
	}
	if answer.StatusCode != http.StatusOK {
		util.ErrorAsJson(w, "Access denied", http.StatusForbidden)
		return false
	}
	return true
}

func answerRedisError(w http.ResponseWriter, description string, err error) error {
	if errors.Is(err, redis.Nil) {
		util.ErrorAsJson(w, "No such key in "+description+" database", http.StatusBadRequest)
	} else if err != nil {
		util.ErrorAsJson(w, "Failed to get from "+description+" database", http.StatusInternalServerError)
	}
	return err
}

func createStock(w http.ResponseWriter, r *http.Request) {
	if !validateAndAnswer(w, r.Header) {
		return
	}

	var stock Stock
	err := json.NewDecoder(r.Body).Decode(&stock)
	if err != nil {
		util.ErrorAsJson(w, "Failed to unmarshal request body", http.StatusBadRequest)
		return
	}

	stock.Code = util.RandomUint64()

	contents, err := json.Marshal(stock)
	if err != nil {
		util.ErrorAsJson(w, "Failed to marshal response body", http.StatusInternalServerError)
		return
	}

	_, err = stocksClient.Set(strconv.FormatUint(stock.Code, 10), contents, 0).Result()
	if err != nil {
		util.ErrorAsJson(w, "Failed to update database", http.StatusInternalServerError)
		return
	}

	_, _ = w.Write(contents)
}

func getAllStocks(w http.ResponseWriter, r *http.Request) {
	if !validateAndAnswer(w, r.Header) {
		return
	}

	keys, err := stocksClient.Keys("*").Result()
	if err != nil {
		util.ErrorAsJson(w, "Failed to get keys from database", http.StatusInternalServerError)
		return
	}

	stocks := make([]Stock, len(keys))
	for i, key := range keys {
		contents, err := stocksClient.Get(key).Result()
		if answerRedisError(w, "stocks", err) != nil {
			return
		}

		var stock Stock
		er := json.Unmarshal([]byte(contents), &stock)
		if er != nil {
			util.ErrorAsJson(w, "Failed to unmarshal data from database", http.StatusInternalServerError)
			return
		}

		stocks[i] = stock
	}

	er := json.NewEncoder(w).Encode(stocks)
	if er != nil {
		util.ErrorAsJson(w, "Failed to marshal response body", http.StatusInternalServerError)
		return
	}
}

func modifyStock(w http.ResponseWriter, r *http.Request) {
	if !validateAndAnswer(w, r.Header) {
		return
	}

	// because of regex in router, key exists in vars
	vars := mux.Vars(r)
	codeString := vars["code"]
	code, err := strconv.ParseUint(codeString, 10, 64)
	if err != nil {
		util.ErrorAsJson(w, "Bad stock number", http.StatusBadRequest)
		return
	}

	var stock Stock
	err = json.NewDecoder(r.Body).Decode(&stock)
	if err != nil {
		util.ErrorAsJson(w, "Failed to unmarshal request body", http.StatusBadRequest)
		return
	}

	stock.Code = code

	contents, err := json.Marshal(stock)
	if err != nil {
		util.ErrorAsJson(w, "Failed to marshal response body", http.StatusInternalServerError)
		return
	}

	_, err = stocksClient.Set(codeString, contents, 0).Result()
	if err != nil {
		util.ErrorAsJson(w, "Failed to update database", http.StatusInternalServerError)
		return
	}

	_, _ = w.Write(contents)
}

func getStock(w http.ResponseWriter, r *http.Request) {
	if !validateAndAnswer(w, r.Header) {
		return
	}

	// because of regex in router, key exists in vars
	vars := mux.Vars(r)
	codeString := vars["code"]

	stock, err := stocksClient.Get(codeString).Result()
	if answerRedisError(w, "stocks", err) != nil {
		return
	}

	_, _ = w.Write([]byte(stock))
}

func deleteStock(w http.ResponseWriter, r *http.Request) {
	if !validateAndAnswer(w, r.Header) {
		return
	}

	// because of regex in router, key exists in vars
	vars := mux.Vars(r)
	codeString := vars["code"]

	_, err := stocksClient.Get(codeString).Result()
	if answerRedisError(w, "stocks", err) != nil {
		return
	}

	_, err = stocksClient.Del(codeString).Result()
	if err != nil {
		util.ErrorAsJson(w, "Failed to delete from database", http.StatusInternalServerError)
		return
	}
}
