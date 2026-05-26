package Receive

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"Synthara-Redux/Globals"
	"Synthara-Redux/Utils"

	"github.com/disgoorg/disgo/voice"
	"github.com/disgoorg/snowflake/v2"
)

// Receiver attaches to a single guild's voice connection and fans Opus frames out to per-user sessions. Implements voice.OpusFrameReceiver
type Receiver struct {

	GuildID snowflake.ID

	dispatcher *Dispatcher

	mu sync.Mutex
	sessions map[snowflake.ID]*Session

	closed bool

}

// VoiceCommandsRequested is true unless VOICE_COMMANDS=false in the environment.
func VoiceCommandsRequested() bool {

	return !strings.EqualFold(os.Getenv("VOICE_COMMANDS"), "false")

}

// IsEnabled returns whether voice command capture should be wired up.
func IsEnabled() bool {

	if !VoiceCommandsRequested() {

		return false

	}

	return WakeWordEnabled()

}

// AttachReceiver hooks a Receiver into the given voice connection.
func AttachReceiver(GuildID snowflake.ID, Conn voice.Conn) *Receiver {

	if Conn == nil {

		return nil

	}

	if !IsEnabled() {

		return nil

	}

	R := &Receiver{

		GuildID: GuildID,

		dispatcher: NewDispatcher(GuildID),
		sessions: make(map[snowflake.ID]*Session),

	}

	Conn.SetOpusFrameReceiver(R)

	return R

}

// Close detaches the receiver and tears down all per-user sessions.
func (R *Receiver) Close() {

	R.mu.Lock()

	if R.closed {

		R.mu.Unlock()
		return

	}

	R.closed = true

	Sessions := make([]*Session, 0, len(R.sessions))

	for _, S := range R.sessions {

		Sessions = append(Sessions, S)

	}

	R.sessions = nil
	R.mu.Unlock()

	for _, S := range Sessions {

		S.Close()

	}

}

// ReceiveOpusFrame implements voice.OpusFrameReceiver. Should always be non-blocking and constant-time.
func (R *Receiver) ReceiveOpusFrame(UserID snowflake.ID, Packet *voice.Packet) error {

	if R == nil || Packet == nil || len(Packet.Opus) == 0 {

		return nil

	}

	// Ignores the bot's own audio...

	if UserID == Globals.DiscordClient.ApplicationID {

		return nil

	}

	if UserID == 0 {

		R.logUnknownSSRC(Packet.SSRC)
		return nil

	}

	Sess := R.getOrCreateSession(UserID)

	if Sess == nil {

		return nil

	}

	Sess.Push(Packet.Opus)
	return nil

}

// CleanupUser should be called when a user stops speaking or leaves the voice channel, to free up resources.
func (R *Receiver) CleanupUser(UserID snowflake.ID) {

	R.mu.Lock()

	Sess, OK := R.sessions[UserID]

	if OK {

		delete(R.sessions, UserID)

	}

	R.mu.Unlock()

	if OK {

		Sess.Close()

	}

}

func (R *Receiver) getOrCreateSession(UserID snowflake.ID) *Session {

	R.mu.Lock()

	if R.closed {

		R.mu.Unlock()
		return nil

	}

	Sess, OK := R.sessions[UserID]

	if OK {

		R.mu.Unlock()
		return Sess

	}

	R.mu.Unlock()

	NewSess, ErrNew := NewSession(R.GuildID, UserID, R.dispatcher)

	if ErrNew != nil {

		Utils.Logger.Error("Receive", fmt.Sprintf("Failed to create session for user %s: %s", UserID, ErrNew.Error()))
		return nil

	}

	R.mu.Lock()

	if R.closed {

		R.mu.Unlock()
		NewSess.Close()
		return nil

	}

	if Existing, ExistsAfterUnlock := R.sessions[UserID]; ExistsAfterUnlock {

		R.mu.Unlock()
		NewSess.Close()

		return Existing

	}

	R.sessions[UserID] = NewSess
	R.mu.Unlock()

	return NewSess

}

var unknownSSRCLogged sync.Map

func (R *Receiver) logUnknownSSRC(SSRC uint32) {

	if _, Loaded := unknownSSRCLogged.LoadOrStore(SSRC, struct{}{}); Loaded {

		return

	}

	Utils.Logger.Warn("Receive", fmt.Sprintf("Opus frame with unknown SSRC %d (guild %s); waiting for Discord speaking event", SSRC, R.GuildID, ))

}
