package main

import (
	"image/png"
	"os"
	"time"

	"gosuda.org/website/ogimage"
)

func main() {
	img := ogimage.GenerateImage("GoSuda", "The 고수다 웹사이트", time.Now())

	f, err := os.Create("ogimage/example/ogimage.png")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	err = png.Encode(f, img)
	if err != nil {
		panic(err)
	}
}
