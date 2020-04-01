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
	http.HandleFunc("/stocks", handler1)

	// Get Stock, Modify Stock, Delete Stock
	http.HandleFunc("/stocks/", handler2)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handler1(w http.ResponseWriter, r *http.Request) {
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

		contents, err = json.Marshal(stock)
		if err != nil {
			http.Error(w, "Failed to marshal response body", http.StatusInternalServerError)
		}

		_, err = client.Set(strconv.FormatUint(stock.Code, 10), contents, 0).Result()
		if err != nil {
			http.Error(w, "Failed to update database", http.StatusInternalServerError)
		}

		_, _ = w.Write(contents)

	} else {
		if r.Method == "" || r.Method == "GET" {
			// Get All Stocks
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
		} else {
			http.Error(w, "Unknown method", http.StatusBadRequest)
		}
	}
}

func handler2(w http.ResponseWriter, r *http.Request) {
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

		contents, err = json.Marshal(stock)
		if err != nil {
			http.Error(w, "Failed to marshal response body", http.StatusInternalServerError)
		}

		_, err = client.Set(strconv.FormatUint(stock.Code, 10), contents, 0).Result()
		if err != nil {
			http.Error(w, "Failed to update database", http.StatusInternalServerError)
		}

		_, _ = w.Write(contents)

	} else {
		if r.Method == "" || r.Method == "GET" {
			// Get Stock
			stock, err := client.Get(strconv.FormatUint(code, 10)).Result()
			if err != nil {
				http.Error(w, "Failed to get from database", http.StatusInternalServerError)
			}
			if stock == "" {
				http.Error(w, "No such stock", http.StatusNotFound)
			}

			_, _ = w.Write([]byte(stock))
		} else {
			if r.Method == "DELETE" {
				// Delete Stock
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
			} else {
				http.Error(w, "Unknown method", http.StatusBadRequest)
			}
		}
	}
}
