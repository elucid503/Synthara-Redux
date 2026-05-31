package Structs

const (

	DefaultVolume = 100
	MinVolume = 0
	MaxVolume = 150
	VolumeStep = 10

)

func ClampVolume(Volume int) int {

	if Volume < MinVolume {

		return MinVolume

	}

	if Volume > MaxVolume {

		return MaxVolume

	}

	return Volume

}

func VolumeProcessingEnabled(Volume int) bool {

	return Volume != DefaultVolume

}

func (G *Guild) SetVolume(Volume int) int {

	G.Features.Volume = ClampVolume(Volume)

	if G.Queue.PlaybackSession != nil && G.Queue.PlaybackSession.Volume != nil {

		G.Queue.PlaybackSession.Volume.SetVolume(G.Features.Volume)

	}

	return G.Features.Volume

}
