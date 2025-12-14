package Audio

// OpusProvider Implements OpusFrameProvider interface for SegmentStreamer
type OpusProvider struct {

	Streamer   *SegmentStreamer
	Segments   []interface{}
	Index      int

}

func (P *OpusProvider) ProvideOpusFrame() ([]byte, error) {

	Frame, Available := P.Streamer.GetNextFrame()

	if Frame != nil && Available {

		return Frame, nil

	}

	return nil, nil

}

func (P *OpusProvider) Close() {

	P.Streamer.Close()

}