package Receive

// PCMBuffer is a fixed-capacity FIFO for little-endian PCM16 chunks.
type PCMBuffer struct {

	data []byte
	cap  int
}

func NewPCMBuffer(maxBytes int) *PCMBuffer {

	return &PCMBuffer{cap: maxBytes}

}

func (B *PCMBuffer) Reset() {

	if B == nil {

		return

	}

	B.data = B.data[:0]

}

func (B *PCMBuffer) Len() int {

	if B == nil {

		return 0

	}

	return len(B.data)

}

func (B *PCMBuffer) Append(Chunk []byte) {

	if B == nil || len(Chunk) == 0 {

		return

	}

	B.data = append(B.data, Chunk...)

	if len(B.data) > B.cap {

		B.data = B.data[len(B.data)-B.cap:]

	}

}

func (B *PCMBuffer) DrainChunks(chunkSize int) [][]byte {

	if B == nil || chunkSize <= 0 {

		return nil

	}

	var Out [][]byte

	for len(B.data) >= chunkSize {

		Out = append(Out, B.data[:chunkSize])
		B.data = B.data[chunkSize:]

	}

	return Out

}

func (B *PCMBuffer) Remainder() []byte {

	if B == nil || len(B.data) == 0 {

		return nil

	}

	Out := make([]byte, len(B.data))
	copy(Out, B.data)
	B.data = B.data[:0]

	return Out

}
