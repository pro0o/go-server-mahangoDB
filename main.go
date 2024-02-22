package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"server/handles"
	"server/types"
	"time"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var client *mongo.Client

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	connectionString := os.Getenv("MONGODB_CONNECTION_STRING")
	if connectionString == "" {
		log.Fatal("MONGODB_CONNECTION_STRING environment variable is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI(connectionString).SetMaxPoolSize(100)
	client, err = mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatal("Error connecting to MongoDB:", err)
	}

	// Ping the MongoDB server to check the connection
	if err := client.Ping(ctx, nil); err != nil {
		log.Fatal("Error pinging MongoDB server:", err)
	}

	log.Println("Connected to worldlink Nepal successfully")
}

func main() {
	router := mux.NewRouter()

	// an instance of App with the MongoDB client
	app := &types.App{Client: client}

	// endpoints
	router.HandleFunc("/api/ocular", func(w http.ResponseWriter, r *http.Request) {
		handles.GetUserData(w, r, app)
	}).Methods("GET")

	router.HandleFunc("/api/ocular", func(w http.ResponseWriter, r *http.Request) {
		handles.PostUserData(w, r, app)
	}).Methods("POST")

	// CORS middleware
	corsHandler := cors.Default().Handler(router)

	log.Println("Server is running on :8080")
	log.Fatal(http.ListenAndServe(":8080", corsHandler))
}
