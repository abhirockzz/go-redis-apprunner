package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
)

var clusterEndpoint string
var username string
var password string

var client *redis.ClusterClient

func init() {
	clusterEndpoint = os.Getenv("MEMORYDB_CLUSTER_ENDPOINT")
	if clusterEndpoint == "" {
		log.Fatal("MEMORYDB_CLUSTER_ENDPOINT env var missing")
	}

	username = os.Getenv("MEMORYDB_USERNAME")
	if username == "" {
		log.Fatal("MEMORYDB_USERNAME env var missing")
	}

	password = os.Getenv("MEMORYDB_PASSWORD")
	if password == "" {
		log.Fatal("MEMORYDB_PASSWORD env var missing")
	}

	log.Println("connecting to redis cluster", clusterEndpoint)

	opts := &redis.ClusterOptions{Username: username, Password: password,
		Addrs:     []string{clusterEndpoint},
		TLSConfig: &tls.Config{},
		//RouteByLatency: true,
		//RouteRandomly: true,
		ReadOnly: true,
	}

	client = redis.NewClusterClient(opts)

	err := client.Ping(context.Background()).Err()
	if err != nil {
		log.Fatalf("failed to connect to memorydb redis. error message - %v", err)
	}

	log.Println("successfully connected to redis cluster", clusterEndpoint)
}

func main() {

	r := mux.NewRouter()

	r.HandleFunc("/", add).Methods(http.MethodPost)
	r.HandleFunc("/{email}", get).Methods(http.MethodGet)

	log.Println("started HTTP server....")
	log.Fatal(http.ListenAndServe(":8080", r))
}

const userHashNamePrefix = "user:"

func add(w http.ResponseWriter, req *http.Request) {

	var user map[string]string
	err := json.NewDecoder(req.Body).Decode(&user)

	if err != nil {
		log.Println("failed to decode json payload", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("user", user)

	userHashName := userHashNamePrefix + user["email"]
	err = client.HSet(req.Context(), userHashName, user).Err()

	if err != nil {
		log.Println("failed to save user", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("added user", userHashName)
}

func get(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	email := vars["email"]

	log.Println("searching for user", email)

	user, err := client.HGetAll(req.Context(), userHashNamePrefix+email).Result()
	if err != nil {
		log.Println("error fetching user", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(user) == 0 {
		log.Println("user not found", email)
		http.Error(w, "user does not exist "+email, http.StatusNotFound)
		return
	}

	err = json.NewEncoder(w).Encode(user)
	if err != nil {
		log.Println("failed to encode user data", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
