package main

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var client *mongo.Client

// data model
type User struct {
	UserName  string `json:"userName,omitempty" bson:"userName,omitempty"`
	ImageData []struct {
		ImageName string `json:"imageName,omitempty" bson:"imageName,omitempty"`
		Image     string `json:"image,omitempty" bson:"image,omitempty"`
		Category  string `json:"category,omitempty" bson:"category,omitempty"`
		Date      string `json:"date,omitempty" bson:"date,omitempty"`
		Saved     bool   `json:"saved,omitempty" bson:"saved,omitempty"`
	} `json:"imageData,omitempty" bson:"imageData,omitempty"`
}

type CustomInfo struct {
	Email       string `json:"email,omitempty" bson:"email,omitempty"`
	CustomImage string `json:"customImage,omitempty" bson:"customImage,omitempty"`
	UserName    string `json:"userName,omitempty" bson:"userName,omitempty"`
}

// Connect to MongoDB through Worldlink Nepal
func init() {
	connectionString := ""
	if connectionString == "" {
		log.Fatal("MONGODB_CONNECTION_STRING environment variable is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI(connectionString).SetMaxPoolSize(100) // Adjust the pool size accordingly.
	var err error
	client, err = mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatal("Error connecting to MongoDB:", err)
	}

	// Ping the MongoDB server to check the connection
	if err := client.Ping(ctx, nil); err != nil {
		log.Fatal("Error pinging MongoDB server:", err)
	}

	log.Println("Connected to MongoDB successfully")
}

func GetUserData(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	// Extract the userName from query parameters
	userName := r.URL.Query().Get("userName")

	if userName == "" {
		http.Error(w, "Bad Request - userName is required in the query parameters", http.StatusBadRequest)
		return
	}

	collection := client.Database("test").Collection("users")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) //donot exceed more than 5sec
	defer cancel()

	// Filter by userName
	filter := bson.M{"userName": userName}

	// Projection to include only necessary fields
	projection := bson.M{"_id": 0, "userName": 1, "imageData": 1}

	cursor, err := collection.Find(ctx, filter, options.Find().SetProjection(projection).SetLimit(10))
	if err != nil {
		handleMongoError(w, userName, err)
		return
	}
	defer cursor.Close(ctx)

	// Use Goroutines for concurrent processing
	ch := make(chan User)
	go processCursor(ctx, cursor, ch)

	var result []User
	for user := range ch {
		result = append(result, user)
	}

	elapsedTime := time.Since(startTime)
	log.Printf("Fetched data for userName %s in %s", userName, elapsedTime)

	respondJSON(w, http.StatusOK, result)
}

func PostUserData(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	// Read the request body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		handleError(w, "Error reading request body", http.StatusInternalServerError)
		return
	}

	// Unmarshal the JSON body into the User struct
	var user User
	if err := json.Unmarshal(body, &user); err != nil {
		handleError(w, "Error decoding request body", http.StatusBadRequest)
		return
	}

	// Use a Goroutine to handle the user concurrently
	go func() {
		handleUserConcurrently(w, user)
	}()

	elapsedTime := time.Since(startTime)
	log.Printf("Post requested data in: %s", elapsedTime)
}

func handleUserConcurrently(w http.ResponseWriter, user User) {
	var wg sync.WaitGroup
	wg.Add(1)
	defer wg.Done()

	collection := client.Database("test").Collection("users")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check if the user already exists
	existingUserEntry := &User{}
	err := collection.FindOne(ctx, bson.M{"userName": user.UserName}).Decode(existingUserEntry)

	if err == nil {
		updateUserEntry(w, collection, user)
	} else if err == mongo.ErrNoDocuments {
		createUserEntry(w, collection, user)
	} else {
		log.Printf("Error fetching user data from MongoDB: %v", err)
	}

	wg.Wait() // Wait for the spawned Goroutine to finish
}

func createUserEntry(w http.ResponseWriter, collection *mongo.Collection, user User) {
	userEntry := &User{
		UserName:  user.UserName,
		ImageData: user.ImageData,
	}

	// Insert the new document into MongoDB
	if _, err := collection.InsertOne(context.Background(), userEntry); err != nil {
		handleMongoError(w, user.UserName, err)
		return
	}

	log.Printf("Success: New user entry created for %s", user.UserName)

	response := map[string]interface{}{"message": "New user entry created"}
	respondJSON(w, http.StatusOK, response)
}

// findImageIndex finds the index of an image with the given name in the ImageData array
func findImageIndex(imageData []struct {
	ImageName string `json:"imageName,omitempty" bson:"imageName,omitempty"`
	Image     string `json:"image,omitempty" bson:"image,omitempty"`
	Category  string `json:"category,omitempty" bson:"category,omitempty"`
	Date      string `json:"date,omitempty" bson:"date,omitempty"`
	Saved     bool   `json:"saved,omitempty" bson:"saved,omitempty"`
}, image string) int {
	for i, img := range imageData {
		if img.Image == image {
			return i
		}
	}
	return -1
}

func updateUserEntry(w http.ResponseWriter, collection *mongo.Collection, user User) {
	// Try to update the existing document in MongoDB
	existingUserEntry := &User{}
	err := collection.FindOne(context.Background(), bson.M{"userName": user.UserName}).Decode(existingUserEntry)

	if err != nil {
		// If there's an error, handle it here
		handleMongoError(w, user.UserName, err)
		return
	}

	// Check if the image URL already exists in the existing user entry
	for _, newImageData := range user.ImageData {
		existingIndex := findImageIndex(existingUserEntry.ImageData, newImageData.Image)

		if existingIndex != -1 {
			// Image with the same name already exists, update the existing entry
			existingUserEntry.ImageData[existingIndex].Image = newImageData.Image
			existingUserEntry.ImageData[existingIndex].Category = newImageData.Category
			existingUserEntry.ImageData[existingIndex].Date = newImageData.Date
			existingUserEntry.ImageData[existingIndex].Saved = newImageData.Saved

			log.Printf("Updated image entry for user %s with URL %s", user.UserName, newImageData.Image)
		} else {
			// Image does not exist, append the new entry to the existing user entry
			existingUserEntry.ImageData = append(existingUserEntry.ImageData, newImageData)
			log.Printf("Appended new image entry for user %s with URL %s", user.UserName, newImageData.Image)
		}
	}

	// Update the existing document in MongoDB
	updateResult, err := collection.UpdateOne(
		context.Background(),
		bson.M{"userName": user.UserName},
		bson.M{"$set": bson.M{"imageData": existingUserEntry.ImageData}},
	)

	if err == nil && updateResult.MatchedCount > 0 {
		log.Printf("Success: Updated user entry for %s", user.UserName)
		response := map[string]interface{}{"message": "User entry updated"}
		respondJSON(w, http.StatusOK, response)
		return
	}

	// If there's an error during the update, handle it here
	handleMongoError(w, user.UserName, err)
}

func respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("Error encoding JSON response: %v", err)
	}
}

func handleMongoError(w http.ResponseWriter, userName string, err error) {
	log.Printf("Error fetching data from MongoDB for userName %s: %v", userName, err)

	// Check if it's a "not found" case
	if err == mongo.ErrNoDocuments {
		http.Error(w, "Not Found - User not found", http.StatusNotFound)
		return
	}

	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}

func handleError(w http.ResponseWriter, errorMsg string, statusCode int) {
	err := errors.New(errorMsg)
	log.Printf("%s: %v", errorMsg, err)
	http.Error(w, errorMsg, statusCode)
}

// processCursor processes the MongoDB cursor concurrently
func processCursor(ctx context.Context, cursor *mongo.Cursor, ch chan<- User) {
	defer close(ch)

	for cursor.Next(ctx) {
		var user User
		if err := cursor.Decode(&user); err != nil {
			log.Printf("Error decoding data: %v", err)
			return
		}
		ch <- user
	}
}

func main() {
	// Initialize router
	router := mux.NewRouter()

	// Define route for requests
	router.HandleFunc("/api/ocular", GetUserData).Methods("GET")
	router.HandleFunc("/api/ocular", PostUserData).Methods("POST")
	router.HandleFunc("/api/customInfo", PostCustomInfo).Methods("POST")

	// CORS middleware
	corsHandler := cors.Default().Handler(router)

	log.Println("Server is running on :8080")
	log.Fatal(http.ListenAndServe(":8080", corsHandler))
}
