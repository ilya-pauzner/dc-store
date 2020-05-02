// +build !solution

package main

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"github.com/go-redis/redis/v7"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"math"
	"math/big"
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

func randomUint64() uint64 {
	bigNumber := math.Pow10(18)
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(bigNumber)))
	return n.Uint64()
}

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

func errorAsJson(w http.ResponseWriter, errorString string, code int) {
	errorMap := make(map[string]string)
	errorMap["error"] = errorString
	errorJson, _ := json.Marshal(errorMap)
	http.Error(w, string(errorJson), code)
}

func validateAndAnswer(w http.ResponseWriter, contents []byte) bool {
	answer, err := http.Post("http://auth:8081/validate", "application/json", bytes.NewReader(contents))
	if err != nil {
		errorAsJson(w, err.Error(), http.StatusInternalServerError)
		return false
	}
	if answer.StatusCode != http.StatusOK {
		errorAsJson(w, "Access denied", http.StatusForbidden)
		return false
	}
	return true
}

func answerRedisError(w http.ResponseWriter, description string, err error) error {
	if errors.Is(err, redis.Nil) {
		errorAsJson(w, "No such key in "+description+" database", http.StatusBadRequest)
	} else if err != nil {
		errorAsJson(w, "Failed to get from "+description+" database", http.StatusInternalServerError)
	}
	return err
}

func createStock(w http.ResponseWriter, r *http.Request) {
	contents, err := ioutil.ReadAll(r.Body)
	if err != nil {
		errorAsJson(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	if !validateAndAnswer(w, contents) {
		return
	}

	var stock Stock
	err = json.Unmarshal(contents, &stock)
	if err != nil {
		errorAsJson(w, "Failed to unmarshal request body", http.StatusBadRequest)
		return
	}

	stock.Code = randomUint64()

	contents, err = json.Marshal(stock)
	if err != nil {
		errorAsJson(w, "Failed to marshal response body", http.StatusInternalServerError)
		return
	}

	_, err = stocksClient.Set(strconv.FormatUint(stock.Code, 10), contents, 0).Result()
	if err != nil {
		errorAsJson(w, "Failed to update database", http.StatusInternalServerError)
		return
	}

	_, _ = w.Write(contents)
}

func getAllStocks(w http.ResponseWriter, r *http.Request) {
	contents, err := ioutil.ReadAll(r.Body)
	if err != nil {
		errorAsJson(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	if !validateAndAnswer(w, contents) {
		return
	}

	keys, err := stocksClient.Keys("*").Result()
	if err != nil {
		errorAsJson(w, "Failed to get keys from database", http.StatusInternalServerError)
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
			errorAsJson(w, "Failed to unmarshal data from database", http.StatusInternalServerError)
			return
		}

		stocks[i] = stock
	}

	er := json.NewEncoder(w).Encode(stocks)
	if er != nil {
		errorAsJson(w, "Failed to marshal response body", http.StatusInternalServerError)
		return
	}
}

func modifyStock(w http.ResponseWriter, r *http.Request) {
	contents, err := ioutil.ReadAll(r.Body)
	if err != nil {
		errorAsJson(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	if !validateAndAnswer(w, contents) {
		return
	}

	// because of regex in router, key exists in vars
	vars := mux.Vars(r)
	codeString := vars["code"]
	code, err := strconv.ParseUint(codeString, 10, 64)
	if err != nil {
		errorAsJson(w, "Bad stock number", http.StatusBadRequest)
		return
	}

	var stock Stock
	err = json.Unmarshal(contents, &stock)
	if err != nil {
		errorAsJson(w, "Failed to unmarshal request body", http.StatusBadRequest)
		return
	}

	stock.Code = code

	contents, err = json.Marshal(stock)
	if err != nil {
		errorAsJson(w, "Failed to marshal response body", http.StatusInternalServerError)
		return
	}

	_, err = stocksClient.Set(codeString, contents, 0).Result()
	if err != nil {
		errorAsJson(w, "Failed to update database", http.StatusInternalServerError)
		return
	}

	_, _ = w.Write(contents)
}

func getStock(w http.ResponseWriter, r *http.Request) {
	contents, err := ioutil.ReadAll(r.Body)
	if err != nil {
		errorAsJson(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	if !validateAndAnswer(w, contents) {
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
	contents, err := ioutil.ReadAll(r.Body)
	if err != nil {
		errorAsJson(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	if !validateAndAnswer(w, contents) {
		return
	}

	// because of regex in router, key exists in vars
	vars := mux.Vars(r)
	codeString := vars["code"]

	_, err = stocksClient.Get(codeString).Result()
	if answerRedisError(w, "stocks", err) != nil {
		return
	}

	_, err = stocksClient.Del(codeString).Result()
	if err != nil {
		errorAsJson(w, "Failed to delete from database", http.StatusInternalServerError)
		return
	}
}
