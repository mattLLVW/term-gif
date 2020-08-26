package models

import (
	"github.com/eliukblau/pixterm/pkg/ansimage"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"log"
	"net/http"
	"os"
	"runtime"
)

// TODO: create custom gif struct and associated methods
// a rendered image extracted from a gif with its timing
type RenderedImg struct {
	Output string
	Delay  int
}

func GetFromUrl(url string) (g *gif.GIF, err error) {
	res, err := http.Get(url)
	if err != nil {
		return g, err
	}
	defer res.Body.Close()
	g, err = gif.DecodeAll(res.Body)
	if err != nil {
		return g, err
	}
	return
}

// Fetch gif on api, render it as ascii and return it
func InsertGif(id string, url string, rev bool) (imgs []RenderedImg, err error) {
	g, err := GetFromUrl(url)
	if err != nil {
		return imgs, err
	}

	stmt, err := db.Prepare("INSERT INTO gif (api_id) VALUES (?)")
	defer stmt.Close()
	if err != nil {
		return imgs, err
	}
	_, err = stmt.Exec(id)
	if err != nil {
		return imgs, err
	}

	imgs = renderGif(g)
	for i, srcImg := range imgs {
		stmt, err := db.Prepare("INSERT INTO gif_data (frame_nb, delay, frame, gif_id) VALUES (?, ?, ?, ?)")
		defer stmt.Close()
		if err != nil {
			return imgs, err
		}
		_, err = stmt.Exec(i, srcImg.Delay, srcImg.Output, id)
		if err != nil {
			return imgs, err
		}
	}
	// Reverse gif
	if rev {
		for i := len(imgs)/2 - 1; i >= 0; i-- {
			opp := len(imgs) - 1 - i
			imgs[i], imgs[opp] = imgs[opp], imgs[i]
		}
	}
	return imgs, nil
}

func GetPreview(url string) (res string) {
	// set image scale factor for ANSIPixel grid, background color and scale mode
	tx, ty := 30, 9
	sfy, sfx := ansimage.BlockSizeY, ansimage.BlockSizeX
	mc := color.RGBA{0x00, 0x00, 0x00, 0xff}
	dm := ansimage.DitheringMode(0)
	sm := ansimage.ScaleMode(2)
	pix, _ := ansimage.NewScaledFromURL(url, sfy*ty, sfx*tx, mc, sm, dm)
	pix.SetMaxProcs(runtime.NumCPU())
	res = pix.Render()
	return
}

// Split gif, transform to ansi code and return a slice of images:delay
func renderGif(g *gif.GIF) (imgs []RenderedImg) {
	// https://stackoverflow.com/a/33296596/8135079
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Error while searching gif %s", r)
		}
	}()

	imgWidth, imgHeight := maxDimensions(g)

	overpaintImage := image.NewRGBA(image.Rect(0, 0, imgWidth, imgHeight))
	draw.Draw(overpaintImage, overpaintImage.Bounds(), g.Image[0], image.ZP, draw.Src)

	// set image scale factor for ANSIPixel grid, background color and scale mode
	tx, ty := 30, 9
	sfy, sfx := ansimage.BlockSizeY, ansimage.BlockSizeX
	mc := color.RGBA{0x00, 0x00, 0x00, 0xff}
	dm := ansimage.DitheringMode(0)
	sm := ansimage.ScaleMode(2)

	for i, srcImg := range g.Image {
		draw.Draw(overpaintImage, overpaintImage.Bounds(), srcImg, image.ZP, draw.Over)
		pix, _ := ansimage.NewScaledFromImage(overpaintImage, sfy*ty, sfx*tx, mc, sm, dm)
		pix.SetMaxProcs(runtime.NumCPU())
		imgs = append(imgs, RenderedImg{Delay: g.Delay[i], Output: pix.Render()})
	}
	return imgs
}

// Get max Gif dimensions.
func maxDimensions(gif *gif.GIF) (x, y int) {
	var lowestX int
	var lowestY int
	var highestX int
	var highestY int

	for _, img := range gif.Image {
		if img.Rect.Min.X < lowestX {
			lowestX = img.Rect.Min.X
		}
		if img.Rect.Min.Y < lowestY {
			lowestY = img.Rect.Min.Y
		}
		if img.Rect.Max.X > highestX {
			highestX = img.Rect.Max.X
		}
		if img.Rect.Max.Y > highestY {
			highestY = img.Rect.Max.Y
		}
	}

	return highestX - lowestX, highestY - lowestY
}

// If anything bad happen, be cute
func OopsGif() []RenderedImg {
	oopsFile, err := os.Open("static/img/oops.gif")
	defer oopsFile.Close()
	if err != nil {
		// Something is really wrong, just stop everything
		panic("can't even open rescue gif")
	}
	g, err := gif.DecodeAll(oopsFile)
	if err != nil {
		// Something is really wrong, just stop everything
		panic("can't even decode rescue gif")
	}
	return renderGif(g)
}
