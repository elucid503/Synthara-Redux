package Structs

import (
	"Synthara-Redux/APIs/Tidal"
	"Synthara-Redux/Globals"
	"context"
	"errors"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (

	MaxSavedQueuesPerGuild = 20
	MaxSavedQueueNameLen   = 32

)

var (

	ErrSavedQueueNameEmpty = errors.New("saved queue name is empty")
	ErrSavedQueueNameTooLong = errors.New("saved queue name is too long")
	ErrSavedQueueLimit = errors.New("saved queue limit reached")
	ErrSavedQueueNotFound = errors.New("saved queue not found")
	ErrSavedQueueEmpty = errors.New("queue has no songs to save")

)

type SavedQueueSnapshot struct {

	Previous []*Tidal.Song `bson:"previous"`
	Current *Tidal.Song `bson:"current,omitempty"`
	Upcoming []*Tidal.Song `bson:"upcoming"`

}

type GuildSavedQueues struct {

	GuildID string `bson:"_id"`
	Queues map[string]SavedQueueSnapshot `bson:"queues"`

}

func NormalizeSavedQueueName(Name string) (string, error) {

	Trimmed := strings.TrimSpace(Name)

	if Trimmed == "" {

		return "", ErrSavedQueueNameEmpty

	}

	if len(Trimmed) > MaxSavedQueueNameLen {

		return "", ErrSavedQueueNameTooLong

	}

	return Trimmed, nil

}

func CloneSong(Song *Tidal.Song) *Tidal.Song {

	if Song == nil {

		return nil

	}

	Copy := *Song
	Copy.Internal = Tidal.SongInternal{}

	return &Copy

}

func CloneSongs(Songs []*Tidal.Song) []*Tidal.Song {

	if len(Songs) == 0 {

		return []*Tidal.Song{}

	}

	Cloned := make([]*Tidal.Song, len(Songs))

	for Index, Song := range Songs {

		Cloned[Index] = CloneSong(Song)

	}

	return Cloned

}

func SnapshotFromQueue(Queue *Queue) (SavedQueueSnapshot, error) {

	if Queue == nil {

		return SavedQueueSnapshot{}, ErrSavedQueueEmpty

	}

	if Queue.Current == nil && len(Queue.Previous) == 0 && len(Queue.Upcoming) == 0 {

		return SavedQueueSnapshot{}, ErrSavedQueueEmpty

	}

	return SavedQueueSnapshot{

		Previous: CloneSongs(Queue.Previous),
		Current: CloneSong(Queue.Current),
		Upcoming: CloneSongs(Queue.Upcoming),

	}, nil

}

func (G *Guild) ApplySavedQueue(Snapshot SavedQueueSnapshot) {

	G.StreamerMutex.Lock()
	defer G.StreamerMutex.Unlock()

	if G.Queue.PlaybackSession != nil {

		G.Queue.PlaybackSession.Stop()
		G.Queue.PlaybackSession = nil

	}

	if G.VoiceMixer != nil {

		G.VoiceMixer.SetSource(nil)

	} else if G.VoiceConnection != nil {

		G.VoiceConnection.SetOpusFrameProvider(nil)

	}

	G.Queue.Previous = CloneSongs(Snapshot.Previous)
	G.Queue.Current = CloneSong(Snapshot.Current)
	G.Queue.Upcoming = CloneSongs(Snapshot.Upcoming)
	G.Queue.Suggestions = []*Tidal.Song{}

	G.Queue.State = StateIdle

	G.Queue.Functions.Updated(&G.Queue)

	if G.Queue.Current != nil {

		go G.Queue.Play()

	}

}

func loadGuildSavedQueues(GuildID string) (*GuildSavedQueues, error) {

	Collection := Globals.Database.Collection("GuildSavedQueues")

	Context, Cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer Cancel()

	Document := &GuildSavedQueues{}

	Error := Collection.FindOne(Context, bson.M{"_id": GuildID}).Decode(Document)

	if Error != nil {

		return &GuildSavedQueues{

			GuildID: GuildID,
			Queues:  make(map[string]SavedQueueSnapshot),

		}, nil

	}

	if Document.Queues == nil {

		Document.Queues = make(map[string]SavedQueueSnapshot)

	}

	return Document, nil

}

func persistGuildSavedQueues(Document *GuildSavedQueues) error {

	Collection := Globals.Database.Collection("GuildSavedQueues")

	Context, Cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer Cancel()

	_, Error := Collection.UpdateOne(

		Context,
		bson.M{"_id": Document.GuildID},
		bson.M{"$set": bson.M{"queues": Document.Queues}},
		options.Update().SetUpsert(true),

	)

	return Error

}

func ListSavedQueueNames(GuildID string) ([]string, error) {

	Document, Error := loadGuildSavedQueues(GuildID)

	if Error != nil {

		return nil, Error

	}

	Names := make([]string, 0, len(Document.Queues))

	for Name := range Document.Queues {

		Names = append(Names, Name)

	}

	return Names, nil

}

func GetSavedQueue(GuildID string, Name string) (*SavedQueueSnapshot, error) {

	Normalized, Error := NormalizeSavedQueueName(Name)

	if Error != nil {

		return nil, Error

	}

	Document, Error := loadGuildSavedQueues(GuildID)

	if Error != nil {

		return nil, Error

	}

	Snapshot, Exists := Document.Queues[Normalized]

	if !Exists {

		return nil, ErrSavedQueueNotFound

	}

	return &Snapshot, nil

}

func SaveGuildQueue(GuildID string, Name string, Snapshot SavedQueueSnapshot) error {

	Normalized, Error := NormalizeSavedQueueName(Name)

	if Error != nil {

		return Error

	}

	Document, Error := loadGuildSavedQueues(GuildID)

	if Error != nil {

		return Error

	}

	_, Exists := Document.Queues[Normalized]

	if !Exists && len(Document.Queues) >= MaxSavedQueuesPerGuild {

		return ErrSavedQueueLimit

	}

	Document.Queues[Normalized] = Snapshot

	return persistGuildSavedQueues(Document)

}

func DeleteSavedQueue(GuildID string, Name string) error {

	Normalized, Error := NormalizeSavedQueueName(Name)

	if Error != nil {

		return Error

	}

	Document, Error := loadGuildSavedQueues(GuildID)

	if Error != nil {

		return Error

	}

	if _, Exists := Document.Queues[Normalized]; !Exists {

		return ErrSavedQueueNotFound

	}

	delete(Document.Queues, Normalized)

	return persistGuildSavedQueues(Document)

}

func SavedQueueSongCount(Snapshot SavedQueueSnapshot) int {

	Count := len(Snapshot.Previous) + len(Snapshot.Upcoming)

	if Snapshot.Current != nil {

		Count++

	}

	return Count

}
