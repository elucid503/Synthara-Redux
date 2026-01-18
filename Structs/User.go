package Structs

import (
	"Synthara-Redux/Globals"
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Favorites Buffer for batching writes

var FavoritesBuffer = make(map[string]map[string]int) // UserID -> URI -> Count
var BufferMutex sync.Mutex
var FlusherRunning = false
var FlusherOnce sync.Once

// Structs

type RecentSearch struct {

	Title string `bson:"title"`
	URI   string `bson:"uri"`

}

type User struct {

	DiscordID string `bson:"_id"` // Primary key
	FirstUse bool `bson:"first_use"` // Indicates if the user is using the bot for the first time
	LastNotificationSeen string `bson:"last_notification_seen,omitempty"` // ID of last seen notification

	RecentSearches []RecentSearch `bson:"recent_searches"`

	Favorites map[string]int `bson:"favorites"` // URI -> Count
	MostRecentMix string `bson:"most_recent_mix,omitempty"` // ID of the last played mix

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
			Favorites:      make(map[string]int),
			MostRecentMix:  "",

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
// SetLastNotificationSeen updates the user's last seen notification ID
func (U *User) SetLastNotificationSeen(NotificationID string) error {

	Collection := Globals.Database.Collection("Users")

	Context, Cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer Cancel()

	U.LastNotificationSeen = NotificationID

	Update := bson.M{

		"$set": bson.M{

			"last_notification_seen": NotificationID,

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

		if Search.URI != URI && Search.Title != Title {

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

// AddFavorite increments the play count for a song URI
func (U *User) AddFavorite(URI string) error {

	// Increment local instance immediately for UI responsiveness if needed

	if U.Favorites == nil {

		U.Favorites = make(map[string]int)

	}

	U.Favorites[URI]++

	// Init flusher continuously to ensure it's running

	FlusherOnce.Do(func() {

		go flushFavoritesLoop()

	})

	// Adds to buffer

	BufferMutex.Lock()
	if FavoritesBuffer[U.DiscordID] == nil {

		FavoritesBuffer[U.DiscordID] = make(map[string]int)

	}

	FavoritesBuffer[U.DiscordID][URI]++
	BufferMutex.Unlock()

	return nil

}

func flushFavoritesLoop() {

	Ticker := time.NewTicker(2 * time.Minute)
	defer Ticker.Stop()

	for range Ticker.C {

		flushFavorites()

	}
}

func flushFavorites() {

	BufferMutex.Lock()

	if len(FavoritesBuffer) == 0 {

		BufferMutex.Unlock()
		return

	}

	// Swap buffer

	CurrentBuffer := FavoritesBuffer
	FavoritesBuffer = make(map[string]map[string]int)
	BufferMutex.Unlock()

	Collection := Globals.Database.Collection("Users")
	var Ops []mongo.WriteModel

	for UserID, Songs := range CurrentBuffer {

		for URI, Count := range Songs {

			// Using dot notation for nested map update in MongoDB
			// Key must not contain dots, which URIs generally don't (they use colons)

			Ops = append(Ops, mongo.NewUpdateOneModel().
				SetFilter(bson.M{"_id": UserID}).
				SetUpdate(bson.M{"$inc": bson.M{fmt.Sprintf("favorites.%s", URI): Count}}).
				SetUpsert(true))

		}

	}

	if len(Ops) > 0 {

		Context, Cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer Cancel()
		
		// Bulk write in batches of 500 if needed, but here simple BulkWrite

		_, Err := Collection.BulkWrite(Context, Ops)

		if Err != nil {

			// In a real app we might want to log this or retry

			fmt.Printf("Error flushing favorites: %v\n", Err)

		}

	}

}

// SetMostRecentMix updates the user's most recent mix ID
func (U *User) SetMostRecentMix(MixID string) error {

	if MixID == "" || U.MostRecentMix == MixID {
		return nil
	}

	U.MostRecentMix = MixID

	Collection := Globals.Database.Collection("Users")

	Context, Cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer Cancel()

	Update := bson.M{
		"$set": bson.M{
			"most_recent_mix": MixID,
		},
	}

	_, UpdateError := Collection.UpdateOne(Context, bson.M{"_id": U.DiscordID}, Update, options.Update().SetUpsert(true))

	return UpdateError

}

// GetTopFavorites returns the top N favorite songs by play count
func (U *User) GetTopFavorites(Limit int) []string {

	type Entry struct {
		URI   string
		Count int
	}

	Entries := make([]Entry, 0, len(U.Favorites))

	for URI, Count := range U.Favorites {

		Entries = append(Entries, Entry{URI, Count})

	}

	sort.Slice(Entries, func(i, j int) bool {

		return Entries[i].Count > Entries[j].Count

	})

	Top := make([]string, 0, Limit)

	for i := 0; i < len(Entries) && i < Limit; i++ {

		Top = append(Top, Entries[i].URI)
		
	}

	return Top

}