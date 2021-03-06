// +build !solution

package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v7"
	"github.com/gorilla/mux"
	"github.com/ilya-pauzner/dc-store/util"
	pb "github.com/ilya-pauzner/dc-store/validator"
	"github.com/streadway/amqp"
	"google.golang.org/grpc"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"
)

type server struct {
	pb.UnimplementedValidatorServer
}

func (s *server) ValidateToken(_ context.Context, req *pb.ValidateRequest) (*pb.ValidateReply, error) {
	log.Printf("Received: %v", req.Token)

	_, err := accessTokenToRefreshTokenClient.Get(req.Token).Result()
	if err != nil {
		errorString, code := util.RedisErrorString("tokens", err)
		if code == http.StatusBadRequest {
			return &pb.ValidateReply{Success: false}, nil
		} else {
			return &pb.ValidateReply{Success: false}, errors.New(errorString)
		}
	}

	if req.Write {
		_, err := accessTokenToAdminClient.Get(req.Token).Result()
		if err != nil {
			errorString, code := util.RedisErrorString("tokens", err)
			if code == http.StatusBadRequest {
				return &pb.ValidateReply{Success: false}, nil
			} else {
				return &pb.ValidateReply{Success: false}, errors.New(errorString)
			}
		}
	}

	return &pb.ValidateReply{Success: true}, nil
}

func startServer() {
	lis, err := net.Listen("tcp", ":8082")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterValidatorServer(s, &server{})
	log.Fatal(s.Serve(lis))
}

var (
	emailToPasswordClient           *redis.Client
	accessTokenToRefreshTokenClient *redis.Client
	refreshTokenToAccessTokenClient *redis.Client

	linkToClickedClient *redis.Client
	emailToLinkClient   *redis.Client

	emailToAdminClient        *redis.Client
	accessTokenToAdminClient  *redis.Client
	refreshTokenToAdminClient *redis.Client

	ch         *amqp.Channel
	emailQueue amqp.Queue
)

func main() {
	emailToPasswordClient = redis.NewClient(&redis.Options{Addr: "db:6379", DB: 1})
	accessTokenToRefreshTokenClient = redis.NewClient(&redis.Options{Addr: "db:6379", DB: 2})
	refreshTokenToAccessTokenClient = redis.NewClient(&redis.Options{Addr: "db:6379", DB: 3})

	linkToClickedClient = redis.NewClient(&redis.Options{Addr: "db:6379", DB: 4})
	emailToLinkClient = redis.NewClient(&redis.Options{Addr: "db:6379", DB: 5})

	emailToAdminClient = redis.NewClient(&redis.Options{Addr: "db:6379", DB: 6})
	accessTokenToAdminClient = redis.NewClient(&redis.Options{Addr: "db:6379", DB: 7})
	refreshTokenToAdminClient = redis.NewClient(&redis.Options{Addr: "db:6379", DB: 8})

	// creating admin
	_ = accessTokenToRefreshTokenClient.Set("root", "toor", 0)
	_ = accessTokenToAdminClient.Set("root", "1", 0)
	_ = refreshTokenToAccessTokenClient.Set("toor", "root", 0)
	_ = refreshTokenToAdminClient.Set("toor", "1", 0)

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

	go startServer()

	r := mux.NewRouter()

	r.HandleFunc("/links/{code:[0-9]+}", activate)

	r.HandleFunc("/register", register).Methods("POST")
	r.HandleFunc("/authorize", authorize).Methods("POST")
	r.HandleFunc("/refresh", refresh).Methods("POST")

	r.HandleFunc("/promote", promote).Methods("POST")

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

	_, err = emailToPasswordClient.Get(email).Result()
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

	_, err = emailToPasswordClient.Set(email, hashedPassword, 0).Result()
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

	hashedPasswordInDataBase, err := emailToPasswordClient.Get(email).Result()
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

	_, err = refreshTokenToAccessTokenClient.Set(refreshTokenString, accessTokenString, 0).Result()
	if err != nil {
		util.ErrorAsJson(w, "Failed to update database", http.StatusInternalServerError)
		return
	}

	_, err = accessTokenToRefreshTokenClient.Set(accessTokenString, refreshTokenString, time.Hour).Result()
	if err != nil {
		util.ErrorAsJson(w, "Failed to update database", http.StatusInternalServerError)
		return
	}

	_, err = emailToAdminClient.Get(email).Result()
	if err == nil {
		// is admin
		_, err = accessTokenToAdminClient.Set(accessTokenString, "1", time.Hour).Result()
		if err != nil {
			util.ErrorAsJson(w, "Failed to update database", http.StatusInternalServerError)
			return
		}

		_, err = refreshTokenToAdminClient.Set(refreshTokenString, "1", 0).Result()
		if err != nil {
			util.ErrorAsJson(w, "Failed to update database", http.StatusInternalServerError)
			return
		}
	} else if err == redis.Nil {
		// is not admin
	} else {
		// redis failed
		_ = util.AnswerRedisError(w, "admins", err)
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

	oldAccessToken, err := refreshTokenToAccessTokenClient.Get(oldRefreshToken).Result()
	if util.AnswerRedisError(w, "refresh_token", err) != nil {
		return
	}

	var admin = false
	_, err = refreshTokenToAdminClient.Get(oldRefreshToken).Result()
	if err == nil {
		// is admin
		admin = true
	} else if err == redis.Nil {
		// is not admin
	} else {
		// redis failed
		_ = util.AnswerRedisError(w, "admins", err)
	}

	_, err = accessTokenToRefreshTokenClient.Del(oldAccessToken).Result()
	if util.AnswerRedisError(w, "access_token", err) != nil {
		return
	}
	if admin {
		_, err = accessTokenToAdminClient.Del(oldAccessToken).Result()
		if util.AnswerRedisError(w, "access_token", err) != nil {
			return
		}
	}

	_, err = refreshTokenToAccessTokenClient.Del(oldRefreshToken).Result()
	if util.AnswerRedisError(w, "refresh_token", err) != nil {
		return
	}
	if admin {
		_, err = refreshTokenToAdminClient.Del(oldRefreshToken).Result()
		if util.AnswerRedisError(w, "refresh_token", err) != nil {
			return
		}
	}

	refreshToken := util.RandomUint64()
	refreshTokenString := strconv.FormatUint(refreshToken, 10)

	accessToken := util.RandomUint64()
	accessTokenString := strconv.FormatUint(accessToken, 10)

	_, err = refreshTokenToAccessTokenClient.Set(refreshTokenString, accessTokenString, 0).Result()
	if util.AnswerRedisError(w, "refresh_token", err) != nil {
		return
	}
	if admin {
		_, err = refreshTokenToAdminClient.Set(refreshTokenString, "1", 0).Result()
		if util.AnswerRedisError(w, "refresh_token", err) != nil {
			return
		}
	}

	_, err = accessTokenToRefreshTokenClient.Set(accessTokenString, refreshTokenString, time.Hour).Result()
	if util.AnswerRedisError(w, "access_token", err) != nil {
		return
	}
	if admin {
		_, err = accessTokenToAdminClient.Set(accessTokenString, "1", time.Hour).Result()
		if util.AnswerRedisError(w, "access_token", err) != nil {
			return
		}
	}

	tokens := make(map[string]string)
	tokens["access_token"] = accessTokenString
	tokens["refresh_token"] = refreshTokenString
	_ = json.NewEncoder(w).Encode(tokens)
}

func promote(w http.ResponseWriter, r *http.Request) {
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

	accessToken := r.Header.Get("access_token")
	if accessToken == "" {
		util.ErrorAsJson(w, "Failed to get access_token from request headers", http.StatusBadRequest)
		return
	}

	_, err = accessTokenToAdminClient.Get(accessToken).Result()
	if err == nil {
		// is admin
		emailToAdminClient.Set(email, "1", 0)
	} else if err == redis.Nil {
		// is not admin
		util.ErrorAsJson(w, "You are not admin", http.StatusForbidden)
		return
	} else {
		// redis failed
		_ = util.AnswerRedisError(w, "admins", err)
		return
	}
}
