package Receive

// opusPreroll keeps recent raw Opus frames while listening.
type opusPreroll struct {

	frames [][]byte
	max int

}

func newOpusPreroll(maxFrames int) opusPreroll {

	return opusPreroll{max: maxFrames}

}

func (P *opusPreroll) Push(Opus []byte) {

	if P == nil || len(Opus) == 0 {

		return

	}

	P.frames = append(P.frames, append([]byte(nil), Opus...))

	if len(P.frames) > P.max {

		P.frames = P.frames[len(P.frames)-P.max:]

	}

}

func (P *opusPreroll) Drain() [][]byte {

	if P == nil || len(P.frames) == 0 {

		return nil

	}

	Out := P.frames
	P.frames = nil

	return Out

}

func (P *opusPreroll) Clear() {

	if P == nil {

		return

	}

	P.frames = nil

}
