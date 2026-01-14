package Structs

import (
	"Synthara-Redux/Globals"
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type RecentSearch struct {

	Title string `bson:"title"`
	URI   string `bson:"uri"`

}

type User struct {

	DiscordID      string         `bson:"_id"` // Primary key
	FirstUse       bool           `bson:"first_use"`

	RecentSearches []RecentSearch `bson:"recent_searches"`

}

// GetUser retrieves a user from MongoDB or creates one if it doesn't exist
func GetUser(DiscordID string) (*User, error) {

	Collection := Globals.Database.Collection("Users")

	Context, Cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer Cancel()

	UserData := &User{}

	Error := Collection.FindOne(Context, bson.M{"_id": DiscordID}).Decode(UserData)

	if Error != nil {

		// Creates New User

		UserData = &User{

			DiscordID:      DiscordID,
			FirstUse:       true,

			RecentSearches: []RecentSearch{},

		}

		_, InsertError := Collection.InsertOne(Context, UserData)

		if InsertError != nil {

			return nil, InsertError

		}

	}

	return UserData, nil

}

func (U *User) SetFirstUse(FirstUse bool) error {

	Collection := Globals.Database.Collection("Users")
	
	Context, Cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer Cancel()

	U.FirstUse = FirstUse

	Update := bson.M{

		"$set": bson.M{

			"first_use": FirstUse,

		},

	}

	_, UpdateError := Collection.UpdateOne(Context, bson.M{"_id": U.DiscordID}, Update, options.Update().SetUpsert(true))

	return UpdateError

}

// AddRecentSearch adds a song to the user's recent searches (max 5, FIFO)
func (U *User) AddRecentSearch(Title string, URI string) error {

	Collection := Globals.Database.Collection("Users")

	Context, Cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer Cancel()

	FilteredSearches := []RecentSearch{}

	for _, Search := range U.RecentSearches {

		if Search.URI != URI {

			FilteredSearches = append(FilteredSearches, Search)
			
		}

	}

	NewSearch := RecentSearch{

		Title: Title,
		URI:   URI,

	}

	FilteredSearches = append([]RecentSearch{NewSearch}, FilteredSearches...)

	if len(FilteredSearches) > 5 {

		FilteredSearches = FilteredSearches[:5]

	}

	U.RecentSearches = FilteredSearches

	Update := bson.M{

		"$set": bson.M{

			"recent_searches": FilteredSearches,
			"first_use":       false,

		},

	}

	_, UpdateError := Collection.UpdateOne(Context, bson.M{"_id": U.DiscordID}, Update, options.Update().SetUpsert(true))

	return UpdateError

}

// ClearRecentSearches removes all recent searches from the user's history
func (U *User) ClearRecentSearches() error {

	Collection := Globals.Database.Collection("Users")

	Context, Cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer Cancel()

	U.RecentSearches = []RecentSearch{}

	Update := bson.M{

		"$set": bson.M{

			"recent_searches": []RecentSearch{},

		},

	}

	_, UpdateError := Collection.UpdateOne(Context, bson.M{"_id": U.DiscordID}, Update, options.Update().SetUpsert(true))

	return UpdateError

}