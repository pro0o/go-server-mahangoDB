package types

import "go.mongodb.org/mongo-driver/mongo"

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

type App struct {
	Client *mongo.Client
}
