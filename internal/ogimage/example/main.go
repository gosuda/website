package main

import (
	"image/png"
	"os"
	"time"

	"gosuda.org/website/internal/ogimage"
)

func main() {
	img := ogimage.GenerateImage("GoSuda", "The 고수다 웹사이트", time.Now())

	f, err := os.Create("internal/ogimage/example/ogimage.png")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	err = png.Encode(f, img)
	if err != nil {
		panic(err)
	}
}
