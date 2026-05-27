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

// Receiver fans Opus frames into per-user voice sessions.
type Receiver struct {

	GuildID snowflake.ID

	dispatcher *Dispatcher

	sessions map[snowflake.ID]*Session

	mu sync.Mutex
	closed bool

}

var (

	receiverRegistry sync.Map
	picoHandlerOnce sync.Once

)

// VoiceCommandsRequested is true unless VOICE_COMMANDS=false.
func VoiceCommandsRequested() bool {

	return !strings.EqualFold(os.Getenv("VOICE_COMMANDS"), "false")

}

// IsEnabled reports whether voice capture should attach to a guild voice connection.
func IsEnabled() bool {

	if !VoiceCommandsRequested() {

		return false

	}

	return WakeDetectorReady()

}

// AttachReceiver hooks a Receiver into the given voice connection.
func AttachReceiver(GuildID snowflake.ID, Conn voice.Conn) *Receiver {

	if Conn == nil || !IsEnabled() {

		return nil

	}

	picoHandlerOnce.Do(func() {

		SetPicoWakeHandler(routePicoWake)

	})

	R := &Receiver{

		GuildID: GuildID,
		dispatcher: NewDispatcher(GuildID),
		sessions: make(map[snowflake.ID]*Session),

	}

	receiverRegistry.Store(GuildID, R)

	Conn.SetOpusFrameReceiver(R)

	Conn.SetEventHandlerFunc(func(_ voice.Gateway, _ voice.Opcode, _ int, Data voice.GatewayMessageData) {

		Speaking, OK := Data.(voice.GatewayMessageDataSpeaking)

		if !OK {

			return

		}

		UserID := Speaking.UserID

		if UserID == 0 {

			UserID = Conn.UserIDBySSRC(Speaking.SSRC)

		}

		if UserID == 0 {

			return

		}

		Active := Speaking.Speaking&voice.SpeakingFlagMicrophone != 0
		R.NotifySpeaking(UserID, Active)

	})

	return R

}

func routePicoWake(StreamID string) {

	GuildID, UserID, OK := parseStreamID(StreamID)

	if !OK {

		return

	}

	Val, Loaded := receiverRegistry.Load(GuildID)

	if !Loaded {

		return

	}

	R, OK := Val.(*Receiver)

	if !OK {

		return

	}

	R.NotifyWake(UserID)

}

// NotifyWake signals a wake-word hit for a user.
func (R *Receiver) NotifyWake(UserID snowflake.ID) {

	R.mu.Lock()
	Sess := R.sessions[UserID]
	R.mu.Unlock()

	if Sess != nil {

		Sess.NotifyWake()

	}

}

// NotifySpeaking updates Discord VAD state for a user session.
func (R *Receiver) NotifySpeaking(UserID snowflake.ID, Active bool) {

	Sess := R.getSession(UserID)

	if Sess != nil {

		Sess.SetDiscordSpeaking(Active)

	}

}

func (R *Receiver) getSession(UserID snowflake.ID) *Session {

	R.mu.Lock()
	Sess := R.sessions[UserID]
	R.mu.Unlock()

	return Sess

}

// Close detaches and tears down all sessions.
func (R *Receiver) Close() {

	receiverRegistry.Delete(R.GuildID)

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

// ReceiveOpusFrame implements voice.OpusFrameReceiver.
func (R *Receiver) ReceiveOpusFrame(UserID snowflake.ID, Packet *voice.Packet) error {

	if R == nil || Packet == nil || len(Packet.Opus) == 0 {

		return nil

	}

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

// CleanupUser frees per-user resources when they leave voice.
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

	if Sess, OK := R.sessions[UserID]; OK {

		R.mu.Unlock()
		return Sess

	}

	R.mu.Unlock()

	if voiceCommandOptOut(UserID) {

		return nil

	}

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

	if Existing, Exists := R.sessions[UserID]; Exists {

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

	Utils.Logger.Warn("Receive", fmt.Sprintf("Opus frame with unknown SSRC %d (guild %s); waiting for Discord speaking event", SSRC, R.GuildID))

}
