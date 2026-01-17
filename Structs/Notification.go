package Structs

import (
	"Synthara-Redux/Globals"
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Notification struct {

	ID          string    `bson:"_id"` // Primary key
	Title       string    `bson:"title"`
	Description string    `bson:"description"`
	Expiry      time.Time `bson:"expiry,omitempty"` // Optional expiration date

}

// GenerateNotificationID generates a random 16 character string
func GenerateNotificationID() string {

	Bytes := make([]byte, 8) // 8 bytes = 16 hex characters
	rand.Read(Bytes)

	return hex.EncodeToString(Bytes)

}

// CreateNotification creates a new notification in the database
func CreateNotification(Title string, Description string, Expiry time.Time) (*Notification, error) {

	Collection := Globals.Database.Collection("Notifications")

	Context, Cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer Cancel()

	NotificationData := &Notification{

		ID:          GenerateNotificationID(),
		Title:       Title,
		Description: Description,
		Expiry:      Expiry,

	}

	_, InsertError := Collection.InsertOne(Context, NotificationData)

	if InsertError != nil {

		return nil, InsertError

	}

	return NotificationData, nil

}

// GetLatestNotification retrieves the most recent notification that hasn't expired
func GetLatestNotification() (*Notification, error) {

	Collection := Globals.Database.Collection("Notifications")

	Context, Cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer Cancel()

	NotificationData := &Notification{}

	// Excludes expired notifications

	Filter := bson.M{

		"$or": []bson.M{

			{"expiry": bson.M{"$exists": false}}, // No expiry
			{"expiry": bson.M{"$gt": time.Now()}}, // Not expired

		},

	}

	// Sorts by ID descending (most recent first)
	
	Options := options.FindOne().SetSort(bson.M{"_id": -1})

	Error := Collection.FindOne(Context, Filter, Options).Decode(NotificationData)

	if Error != nil {

		return nil, Error

	}

	return NotificationData, nil

}

// GetNotificationByID retrieves a notification by its ID
func GetNotificationByID(ID string) (*Notification, error) {

	Collection := Globals.Database.Collection("Notifications")

	Context, Cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer Cancel()

	NotificationData := &Notification{}

	Error := Collection.FindOne(Context, bson.M{"_id": ID}).Decode(NotificationData)

	if Error != nil {

		return nil, Error

	}

	return NotificationData, nil

}