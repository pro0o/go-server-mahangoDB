// File: handles.go

package handles

import (
	"context"
	"log"
	"net/http"
	"time"

	"server/types"
	"server/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func GetUserData(w http.ResponseWriter, r *http.Request, app *types.App) {
	startTime := time.Now()

	userName := r.URL.Query().Get("userName")
	if userName == "" {
		http.Error(w, "Bad Request - userName is required in the query parameters", http.StatusBadRequest)
		return
	}

	collection := app.Client.Database("test").Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Filter the bson by userName
	filter := bson.M{"userName": userName}

	// Projection to include only necessary fields
	projection := bson.M{"_id": 0, "userName": 1, "imageData": 1}
	cursor, err := collection.Find(ctx, filter, options.Find().SetProjection(projection).SetLimit(10))
	if err != nil {
		utils.HandleMongoError(w, userName, err)
		return
	}
	defer cursor.Close(ctx)

	ch := make(chan types.User)
	go utils.ProcessCursor(ctx, cursor, ch)

	var result []types.User
	for user := range ch {
		result = append(result, user)
	}

	elapsedTime := time.Since(startTime)
	log.Printf("Fetched data for userName %s in %s", userName, elapsedTime)

	utils.RespondJSON(w, http.StatusOK, result)
}
