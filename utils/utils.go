package utils

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"server/types"

	"go.mongodb.org/mongo-driver/mongo"
)

func RespondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("Error encoding JSON response: %v", err)
	}
}

func HandleMongoError(w http.ResponseWriter, userName string, err error) {
	log.Printf("Error fetching data from MongoDB for userName %s: %v", userName, err)

	if err == mongo.ErrNoDocuments {
		http.Error(w, "Not Found - User not found", http.StatusNotFound)
		return
	}
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}

func HandleError(w http.ResponseWriter, errorMsg string, statusCode int) {
	err := errors.New(errorMsg)
	log.Printf("%s: %v", errorMsg, err)
	http.Error(w, errorMsg, statusCode)
}

func ProcessCursor(ctx context.Context, cursor *mongo.Cursor, ch chan<- types.User) {
	defer close(ch)

	for cursor.Next(ctx) {
		var user types.User
		if err := cursor.Decode(&user); err != nil {
			log.Printf("Error decoding data: %v", err)
			return
		}
		ch <- user
	}
}
