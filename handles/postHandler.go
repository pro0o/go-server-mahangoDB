package handles

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"server/types"
	"server/utils"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func createUserEntry(w http.ResponseWriter, collection *mongo.Collection, user types.User) {
	userEntry := &types.User{
		UserName:  user.UserName,
		ImageData: user.ImageData,
	}

	// Insert the new document into MongoDB
	if _, err := collection.InsertOne(context.Background(), userEntry); err != nil {
		utils.HandleMongoError(w, user.UserName, err)
		return
	}

	log.Printf("Success: New user entry created for %s", user.UserName)

	response := map[string]interface{}{"message": "New user entry created"}
	utils.RespondJSON(w, http.StatusOK, response)
}

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

func updateUserEntry(w http.ResponseWriter, collection *mongo.Collection, user types.User) {
	// Try to update the existing document in MongoDB
	existingUserEntry := &types.User{}
	err := collection.FindOne(context.Background(), bson.M{"userName": user.UserName}).Decode(existingUserEntry)

	if err != nil {
		utils.HandleMongoError(w, user.UserName, err)
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
		utils.RespondJSON(w, http.StatusOK, response)
		return
	}

	utils.HandleMongoError(w, user.UserName, err)
}

func handleUserConcurrently(w http.ResponseWriter, app *types.App, user types.User) {
	var wg sync.WaitGroup
	wg.Add(1)
	defer wg.Done()

	collection := app.Client.Database("test").Collection("users")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	existingUserEntry := &types.User{}
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

func PostUserData(w http.ResponseWriter, r *http.Request, app *types.App) {
	startTime := time.Now()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		utils.HandleError(w, "Error reading request body", http.StatusInternalServerError)
		return
	}

	// Unmarshal the JSON body into the User struct
	var user types.User
	if err := json.Unmarshal(body, &user); err != nil {
		utils.HandleError(w, "Error decoding request body", http.StatusBadRequest)
		return
	}

	go func() {
		handleUserConcurrently(w, app, user)
	}()

	elapsedTime := time.Since(startTime)
	log.Printf("Post requested data in: %s", elapsedTime)
}
