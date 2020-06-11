// +build !solution

package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v7"
	"github.com/gorilla/mux"
	"github.com/ilya-pauzner/dc-store/util"
	"github.com/streadway/amqp"
	"log"
	"net/http"
	"strconv"
	"time"
)

var (
	passwordsClient     *redis.Client
	accessTokensClient  *redis.Client
	refreshTokensClient *redis.Client
	linkToClickedClient *redis.Client
	emailToLinkClient   *redis.Client

	ch         *amqp.Channel
	emailQueue amqp.Queue
)

func main() {
	passwordsClient = redis.NewClient(&redis.Options{Addr: "db:6379", DB: 1})
	accessTokensClient = redis.NewClient(&redis.Options{Addr: "db:6379", DB: 2})
	refreshTokensClient = redis.NewClient(&redis.Options{Addr: "db:6379", DB: 3})
	linkToClickedClient = redis.NewClient(&redis.Options{Addr: "db:6379", DB: 4})
	emailToLinkClient = redis.NewClient(&redis.Options{Addr: "db:6379", DB: 5})

	conn, err := amqp.Dial("amqp://guest:guest@rabbitmq:5672/")
	if err != nil {
		log.Fatalf("%s: %s", "Failed to connect to RabbitMQ", err)
	}
	defer func() { _ = conn.Close() }()

	ch, err = conn.Channel()
	if err != nil {
		log.Fatalf("%s: %s", "Failed to open a channel", err)
	}
	defer func() { _ = ch.Close() }()

	emailQueue, err = ch.QueueDeclare(
		"email", // name
		false,   // durable
		false,   // delete when unused
		false,   // exclusive
		false,   // no-wait
		nil,     // arguments
	)
	if err != nil {
		log.Fatalf("%s: %s", "Failed to declare a queue", err)
	}

	r := mux.NewRouter()

	r.HandleFunc("/links/{code:[0-9]+}", activate)

	r.HandleFunc("/register", register).Methods("POST")
	r.HandleFunc("/authorize", authorize).Methods("POST")
	r.HandleFunc("/refresh", refresh).Methods("POST")
	r.HandleFunc("/validate", validate).Methods("POST")

	log.Fatal(http.ListenAndServe(":8081", r))
}

func sendMessageToQueue(message []byte) error {
	return ch.Publish(
		"",              // exchange
		emailQueue.Name, // routing key
		false,           // mandatory
		false,           // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        message,
		})
}

func activate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	codeString := vars["code"]

	value, err := linkToClickedClient.Get(codeString).Result()
	if util.AnswerRedisError(w, "activation links", err) != nil {
		return
	}

	if value != "0" {
		util.ErrorAsJson(w, "Activation link already used", http.StatusBadRequest)
		return
	}
	value = "1"

	_, err = linkToClickedClient.Set(codeString, value, 0).Result()
	if err != nil {
		util.ErrorAsJson(w, "Failed to update database", http.StatusInternalServerError)
		return
	}
}

func register(w http.ResponseWriter, r *http.Request) {
	var data map[string]string
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		util.ErrorAsJson(w, "Failed to unmarshal request body", http.StatusBadRequest)
		return
	}

	email, ok := data["email"]
	if !ok {
		util.ErrorAsJson(w, "Failed to get email from request body", http.StatusBadRequest)
		return
	}

	_, err = passwordsClient.Get(email).Result()
	if err == nil {
		util.ErrorAsJson(w, "email already exists", http.StatusBadRequest)
		return
	} else if !errors.Is(err, redis.Nil) {
		util.ErrorAsJson(w, "Failed to get from database", http.StatusInternalServerError)
		return
	}

	password, ok := data["password"]
	if !ok {
		util.ErrorAsJson(w, "Failed to get password from request body", http.StatusBadRequest)
		return
	}

	hash := sha256.New()
	hashedPassword := hash.Sum([]byte(email + password))

	_, err = passwordsClient.Set(email, hashedPassword, 0).Result()
	if err != nil {
		util.ErrorAsJson(w, "Failed to update database", http.StatusInternalServerError)
		return
	}

	linkCode := strconv.FormatUint(util.RandomUint64(), 10)
	_, err = linkToClickedClient.Set(linkCode, "0", 0).Result()
	if err != nil {
		util.ErrorAsJson(w, "Failed to update database", http.StatusInternalServerError)
		return
	}
	_, err = emailToLinkClient.Set(email, linkCode, 0).Result()
	if err != nil {
		util.ErrorAsJson(w, "Failed to update database", http.StatusInternalServerError)
		return
	}

	link := fmt.Sprintf("localhost:8081/links/%s", linkCode)
	err = sendMessageToQueue([]byte(link))
	if err != nil {
		util.ErrorAsJson(w, "Failed to send message", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func authorize(w http.ResponseWriter, r *http.Request) {
	var data map[string]string
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		util.ErrorAsJson(w, "Failed to unmarshal request body", http.StatusBadRequest)
		return
	}

	email, ok := data["email"]
	if !ok {
		util.ErrorAsJson(w, "Failed to get email from request body", http.StatusBadRequest)
		return
	}

	linkCode, err := emailToLinkClient.Get(email).Result()
	if util.AnswerRedisError(w, "registered emails", err) != nil {
		return
	}

	activated, err := linkToClickedClient.Get(linkCode).Result()
	if util.AnswerRedisError(w, "registered emails", err) != nil {
		return
	}
	if activated != "1" {
		util.ErrorAsJson(w, "Email-password pair not activated yet", http.StatusForbidden)
		return
	}

	password, ok := data["password"]
	if !ok {
		util.ErrorAsJson(w, "Failed to get password from request body", http.StatusBadRequest)
		return
	}

	hash := sha256.New()
	hashedPassword := hash.Sum([]byte(email + password))

	hashedPasswordInDataBase, err := passwordsClient.Get(email).Result()
	if util.AnswerRedisError(w, "registered emails", err) != nil {
		return
	}
	if !bytes.Equal(hashedPassword, []byte(hashedPasswordInDataBase)) {
		util.ErrorAsJson(w, "Wrong password", http.StatusForbidden)
		return
	}

	tokens := make(map[string]string)

	refreshToken := util.RandomUint64()
	refreshTokenString := strconv.FormatUint(refreshToken, 10)
	tokens["refresh_token"] = refreshTokenString

	accessToken := util.RandomUint64()
	accessTokenString := strconv.FormatUint(accessToken, 10)
	tokens["access_token"] = accessTokenString

	_, err = refreshTokensClient.Set(refreshTokenString, accessTokenString, 0).Result()
	if err != nil {
		util.ErrorAsJson(w, "Failed to update database", http.StatusInternalServerError)
		return
	}

	_, err = accessTokensClient.Set(accessTokenString, refreshTokenString, time.Hour).Result()
	if err != nil {
		util.ErrorAsJson(w, "Failed to update database", http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(w).Encode(tokens)
	if err != nil {
		util.ErrorAsJson(w, "Failed to marshal response body", http.StatusInternalServerError)
		return
	}
}

func refresh(w http.ResponseWriter, r *http.Request) {
	oldRefreshToken := r.Header.Get("refresh_token")
	if oldRefreshToken == "" {
		util.ErrorAsJson(w, "Failed to get access_token from request headers", http.StatusBadRequest)
		return
	}

	oldAccessToken, err := refreshTokensClient.Get(oldRefreshToken).Result()
	if util.AnswerRedisError(w, "refresh_token", err) != nil {
		return
	}

	_, err = accessTokensClient.Del(oldAccessToken).Result()
	if util.AnswerRedisError(w, "access_token", err) != nil {
		return
	}

	_, err = refreshTokensClient.Del(oldRefreshToken).Result()
	if util.AnswerRedisError(w, "refresh_token", err) != nil {
		return
	}

	refreshToken := util.RandomUint64()
	refreshTokenString := strconv.FormatUint(refreshToken, 10)

	accessToken := util.RandomUint64()
	accessTokenString := strconv.FormatUint(accessToken, 10)

	_, err = refreshTokensClient.Set(refreshTokenString, accessTokenString, 0).Result()
	if util.AnswerRedisError(w, "refresh_token", err) != nil {
		return
	}

	_, err = accessTokensClient.Set(accessTokenString, refreshTokenString, time.Hour).Result()
	if util.AnswerRedisError(w, "access_token", err) != nil {
		return
	}

	tokens := make(map[string]string)
	tokens["access_token"] = accessTokenString
	tokens["refresh_token"] = refreshTokenString
	_ = json.NewEncoder(w).Encode(tokens)
}

func validate(w http.ResponseWriter, r *http.Request) {
	accessTokenString := r.Header.Get("access_token")
	if accessTokenString == "" {
		util.ErrorAsJson(w, "Failed to get access_token from request headers", http.StatusBadRequest)
		return
	}

	_, err := accessTokensClient.Get(accessTokenString).Result()
	if util.AnswerRedisError(w, "access_token", err) != nil {
		return
	}

	w.WriteHeader(http.StatusOK)
}
