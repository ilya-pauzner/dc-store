// +build !solution

package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

type Stock struct {
	Name       string   `"json"`
	Code       uint64   `"json, omitempty"`
	Categories []string `"json"`
}

var (
	stocks map[uint64]Stock
	lock   sync.Mutex
)

func main() {
	stocks = make(map[uint64]Stock)

	// Create Stock, Get All Stocks
	http.HandleFunc("/stocks", handler1)

	// Get Stock, Modify Stock, Delete Stock
	http.HandleFunc("/stocks/", handler2)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handler1(w http.ResponseWriter, r *http.Request) {
	lock.Lock()
	defer lock.Unlock()

	if r.Method == "POST" {
		// Create Stock
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
		stocks[stock.Code] = stock

		contents, err = json.Marshal(stock)
		if err != nil {
			http.Error(w, "Failed to marshal response body", http.StatusInternalServerError)
		}

		w.Write(contents)

	} else {
		if r.Method == "" || r.Method == "GET" {
			// Get All Stocks
			contents, err := json.Marshal(stocks)
			if err != nil {
				http.Error(w, "Failed to marshal response body", http.StatusInternalServerError)
			}

			w.Write(contents)
		} else {
			http.Error(w, "Unknown method", http.StatusBadRequest)
		}
	}
}

func handler2(w http.ResponseWriter, r *http.Request) {
	lock.Lock()
	defer lock.Unlock()

	parts := strings.Split(r.URL.Path, "/")
	key := parts[len(parts)-1]

	code, err := strconv.ParseUint(key, 10, 64)
	if err != nil {
		http.Error(w, "Bad stock number", http.StatusBadRequest)
	}

	if r.Method == "PUT" {
		// Modify Stock
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
		stocks[stock.Code] = stock

		contents, err = json.Marshal(stock)
		if err != nil {
			http.Error(w, "Failed to marshal response body", http.StatusInternalServerError)
		}

		w.Write(contents)

	} else {
		if r.Method == "" || r.Method == "GET" {
			// Get Stock
			stock, ok := stocks[code]
			if !ok {
				http.Error(w, "No such stock", http.StatusNotFound)
			}

			contents, err := json.Marshal(stock)
			if err != nil {
				http.Error(w, "Failed to marshal response body", http.StatusInternalServerError)
			}

			w.Write(contents)
		} else {
			if r.Method == "DELETE" {
				// Delete Stock
				_, ok := stocks[code]
				if !ok {
					http.Error(w, "No such stock", http.StatusNotFound)
				}

				delete(stocks, code)
			} else {
				http.Error(w, "Unknown method", http.StatusBadRequest)
			}
		}
	}
}
