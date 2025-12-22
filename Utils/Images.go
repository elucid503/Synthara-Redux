package Utils

import (
	"fmt"
	"image"
	"net/http"

	_ "image/jpeg" // register JPEG decoder
	_ "image/png"  // register PNG decoder

	"github.com/cenkalti/dominantcolor"
)

// GetDominantColorHex fetches (and decodes) a PNG/JPEG image from a URL and returns the dominant color as a hex string
func GetDominantColorHex(ImageURL string) (int, error) {
		
	ImageResp, ReqError := http.Get(ImageURL)

	if ReqError != nil {

		return 0xFFFFFF, fmt.Errorf("failed to fetch image: %w", ReqError)

	}

	defer ImageResp.Body.Close()

	if ImageResp.StatusCode != http.StatusOK {

		return 0xFFFFFF, fmt.Errorf("failed to fetch image: status code %d", ImageResp.StatusCode)
		
	}

	ImageBytes, _, ReqError := image.Decode(ImageResp.Body)

	if ReqError != nil {

		return 0xFFFFFF, fmt.Errorf("failed to decode image: %w", ReqError)

	}

	DominantColorRGB := dominantcolor.Find(ImageBytes)

	HexValue := (int(DominantColorRGB.R) << 16) + (int(DominantColorRGB.G) << 8) + int(DominantColorRGB.B) // Converts RGB to Hex by shifting 16, 8, and 0 bits

	return HexValue, nil

}