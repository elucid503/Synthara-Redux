package Receive

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"

	"Synthara-Redux/Utils"

	"github.com/disgoorg/snowflake/v2"
)

const (

	picoOpOpen = 1
	picoOpPCM = 2
	picoOpClose = 3
	picoOpWake = 4

	defaultPicoDir = "./Modules/pico"

)

var (

	picoProcess *exec.Cmd
	picoReady atomic.Bool

	picoStdin io.WriteCloser

	picoInitOnce sync.Once
	picoInitErr error

	picoWriteMu sync.Mutex

	picoWakeHandler func(streamID string)

)

// WakeDetectorReady reports whether the Porcupine sidecar is running.
func WakeDetectorReady() bool {

	picoInitOnce.Do(startPicoProcess)
	return picoInitErr == nil && picoReady.Load()

}

func streamID(GuildID, UserID snowflake.ID) string {

	return fmt.Sprintf("%d:%d", GuildID, UserID)

}

func picoRunner() string {

	if Runner := os.Getenv("PICO_RUNNER"); Runner != "" {

		return Runner

	}

	return "node"

}

func picoBuild(Dir string) error {

	Cmd := exec.Command("npm", "run", "build")
	Cmd.Dir = Dir
	Cmd.Stderr = os.Stderr

	return Cmd.Run()

}

// picoArgs returns node arguments for the sidecar. Assumes dist/index.js is already built relative to Dir (call picoBuild first if not).
func picoArgs(Dir string) []string {

	if Script := os.Getenv("PICO_SCRIPT"); Script != "" {

		return []string{Script}

	}

	return []string{"dist/index.js"}

}

func startPicoProcess() {

	Dir := os.Getenv("PICO_DIR")

	if Dir == "" {

		Dir = defaultPicoDir

	}

	Model := filepath.Join(Dir, "model", "synthara.ppn")

	if _, ErrStat := os.Stat(Model); ErrStat != nil {

		picoInitErr = fmt.Errorf("porcupine model: %w", ErrStat)
		Utils.Logger.Warn("Receive", "Wake detector disabled: "+picoInitErr.Error())

		return

	}

	if os.Getenv("PICO_SCRIPT") == "" {

		Built := filepath.Join(Dir, "dist", "index.js")

		if _, ErrStat := os.Stat(Built); ErrStat != nil {

			Utils.Logger.Info("Receive", "Building Porcupine sidecar (dist not found)...")

			if ErrBuild := picoBuild(Dir); ErrBuild != nil {

				picoInitErr = fmt.Errorf("pico build: %w", ErrBuild)
				Utils.Logger.Warn("Receive", "Wake detector disabled: "+picoInitErr.Error())

				return

			}

		}

	}

	Cmd := exec.Command(picoRunner(), picoArgs(Dir)...)
	Cmd.Dir = Dir

	Stdin, ErrStdin := Cmd.StdinPipe()

	if ErrStdin != nil {

		picoInitErr = ErrStdin
		return

	}

	Stdout, ErrStdout := Cmd.StdoutPipe()

	if ErrStdout != nil {

		picoInitErr = ErrStdout
		return

	}

	Cmd.Stderr = os.Stderr

	if ErrStart := Cmd.Start(); ErrStart != nil {

		picoInitErr = ErrStart
		Utils.Logger.Warn("Receive", "Wake detector disabled: "+ErrStart.Error())

		return

	}

	picoProcess = Cmd
	picoStdin = Stdin
	picoReady.Store(true)

	go readPicoWake(Stdout)

	go func() {

		if ErrWait := Cmd.Wait(); ErrWait != nil && !errors.Is(ErrWait, os.ErrProcessDone) {

			Utils.Logger.Warn("Receive", "Porcupine sidecar exited: "+ErrWait.Error())

		}

		picoReady.Store(false)

	}()

}

func readPicoWake(Out io.Reader) {

	Buf := make([]byte, 0, 512)
	ReadBuf := make([]byte, 4096)

	for {

		N, ErrRead := Out.Read(ReadBuf)

		if N > 0 {

			Buf = append(Buf, ReadBuf[:N]...)
			Buf = drainPicoWake(Buf)

		}

		if ErrRead != nil {

			return

		}

	}

}

func drainPicoWake(Buf []byte) []byte {

	for len(Buf) >= 3 {

		if Buf[0] != picoOpWake {

			return Buf

		}

		IDLen := int(binary.BigEndian.Uint16(Buf[1:3]))
		FrameLen := 3 + IDLen

		if len(Buf) < FrameLen {

			return Buf

		}

		ID := string(Buf[3:FrameLen])
		Buf = Buf[FrameLen:]

		if Fn := picoWakeHandler; Fn != nil {

			Fn(ID)

		}

	}

	return Buf

}

func picoWriteFrame(Op byte, StreamID string, PCM []byte) error {

	if !picoReady.Load() || picoStdin == nil {

		return errors.New("porcupine sidecar not ready")

	}

	ID := []byte(StreamID)
	FrameLen := 3 + len(ID)

	if Op == picoOpPCM {

		FrameLen += 4 + len(PCM)

	}

	Frame := make([]byte, FrameLen)
	Frame[0] = Op

	binary.BigEndian.PutUint16(Frame[1:3], uint16(len(ID)))

	copy(Frame[3:], ID)

	if Op == picoOpPCM {

		Off := 3 + len(ID)
		binary.BigEndian.PutUint32(Frame[Off:Off+4], uint32(len(PCM)))
		copy(Frame[Off+4:], PCM)

	}

	picoWriteMu.Lock()
	_, Err := picoStdin.Write(Frame)
	picoWriteMu.Unlock()

	return Err

}

// PicoOpenStream registers a user stream with the sidecar.
func PicoOpenStream(GuildID, UserID snowflake.ID) error {

	picoInitOnce.Do(startPicoProcess)

	return picoWriteFrame(picoOpOpen, streamID(GuildID, UserID), nil)

}

// PicoCloseStream tears down a user stream in the sidecar.
func PicoCloseStream(GuildID, UserID snowflake.ID) error {

	if !picoReady.Load() {

		return nil

	}

	return picoWriteFrame(picoOpClose, streamID(GuildID, UserID), nil)

}

// PicoFeedPCM sends 16 kHz mono PCM to the wake detector for one stream.
func PicoFeedPCM(GuildID, UserID snowflake.ID, PCM []byte) error {

	if len(PCM) == 0 {

		return nil

	}

	return picoWriteFrame(picoOpPCM, streamID(GuildID, UserID), PCM)

}

// SetPicoWakeHandler registers the global wake callback (stream id guild:user).
func SetPicoWakeHandler(fn func(streamID string)) {

	picoWakeHandler = fn

}
