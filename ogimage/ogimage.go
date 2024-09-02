// Package ogimage provides functionality for generating Open Graph images.
package ogimage

import (
	"image"
	"sync"
	"time"

	"github.com/fogleman/gg"
	"golang.org/x/image/font"
)

// font_pool_IBMPlexSansKR_Medium is a sync.Pool for caching IBM Plex Mono Medium font faces.
var font_pool_IBMPlexSansKR_Medium = sync.Pool{
	New: func() interface{} {
		f, err := gg.LoadFontFace("fonts/IBMPlexSansKR-Medium.ttf", 100)
		if err != nil {
			panic(err)
		}
		return f
	},
}

// font_pool_IBMPlexSansKR_Thin is a sync.Pool for caching IBM Plex Mono Thin font faces.
var font_pool_IBMPlexSansKR_Thin = sync.Pool{
	New: func() interface{} {
		f, err := gg.LoadFontFace("fonts/IBMPlexSansKR-Thin.ttf", 60)
		if err != nil {
			panic(err)
		}
		return f
	},
}

// GenerateImage creates an Open Graph image with the given logo, title, and date.
func GenerateImage(name string, title string, date time.Time) (image.Image, error) {
	// Create a new context with the specified dimensions
	ctx := gg.NewContext(1150, 630)
	ctx.Clear()

	// Set background color
	ctx.SetRGBA255(0xf2, 0xe7, 0xd5, 0xff)
	ctx.Clear()

	// Draw inner rectangle
	ctx.SetRGBA255(0x05, 0x15, 0x2a, 0xff)
	ctx.DrawRectangle(20, 20, 1150-40, 630-40)
	ctx.Fill()

	// Get and set IBM Plex Mono Medium font
	_font_IBMPlexMono_Medium := font_pool_IBMPlexSansKR_Medium.Get().(font.Face)
	defer font_pool_IBMPlexSansKR_Medium.Put(_font_IBMPlexMono_Medium)
	ctx.SetFontFace(_font_IBMPlexMono_Medium)

	// Draw title
	ctx.SetRGBA255(0xf2, 0xe7, 0xd5, 0xff)
	ctx.DrawStringWrapped(title, 40, 50, 0, 0, 1150-80, 1.5, gg.AlignLeft)

	// Get and set IBM Plex Mono Thin font
	_font_IBMPlexMono_Thin := font_pool_IBMPlexSansKR_Thin.Get().(font.Face)
	defer font_pool_IBMPlexSansKR_Thin.Put(_font_IBMPlexMono_Thin)
	ctx.SetFontFace(_font_IBMPlexMono_Thin)

	// Draw date
	d := date.Format("2006-01-02")
	tx, _ := ctx.MeasureString(d)
	ctx.DrawString(d, 1150-40-tx, 630-40)

	// Draw logo
	ctx.DrawString(name, 40, 630-40)

	// Return the generated image
	return ctx.Image(), nil
}
