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

// a rendered image extracted from a gif with its timing
type RenderedImg struct {
	Output string
	Delay  int
}

type AnsiGif struct {
	Gif      *gif.GIF
	Rendered []RenderedImg
}

func (g *AnsiGif) Get(url string) (err error) {
	res, err := http.Get(url)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	g.Gif, err = gif.DecodeAll(res.Body)
	if err != nil {
		return err
	}
	return nil
}

func (g *AnsiGif) Render() {
	// https://stackoverflow.com/a/33296596/8135079
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Error while searching gif %s", r)
		}
	}()

	imgWidth, imgHeight := g.maxDimensions()

	overpaintImage := image.NewRGBA(image.Rect(0, 0, imgWidth, imgHeight))
	draw.Draw(overpaintImage, overpaintImage.Bounds(), g.Gif.Image[0], image.ZP, draw.Src)

	// set image scale factor for ANSIPixel grid, background color and scale mode
	tx, ty := 30, 9
	sfy, sfx := ansimage.BlockSizeY, ansimage.BlockSizeX
	mc := color.RGBA{0x00, 0x00, 0x00, 0xff}
	dm := ansimage.DitheringMode(0)
	sm := ansimage.ScaleMode(2)

	for i, srcImg := range g.Gif.Image {
		draw.Draw(overpaintImage, overpaintImage.Bounds(), srcImg, image.ZP, draw.Over)
		pix, _ := ansimage.NewScaledFromImage(overpaintImage, sfy*ty, sfx*tx, mc, sm, dm)
		pix.SetMaxProcs(runtime.NumCPU())
		g.Rendered = append(g.Rendered, RenderedImg{Delay: g.Gif.Delay[i], Output: pix.Render()})
	}
}

// Get max AnsiGif dimensions.
func (g *AnsiGif) maxDimensions() (x, y int) {
	var lowestX int
	var lowestY int
	var highestX int
	var highestY int

	for _, img := range g.Gif.Image {
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
func (g *AnsiGif) Oops() {
	oopsFile, err := os.Open("static/img/oops.gif")
	defer oopsFile.Close()
	if err != nil {
		// Something is really wrong, just stop everything
		panic("can't even open rescue gif")
	}
	g.Gif, err = gif.DecodeAll(oopsFile)
	if err != nil {
		// Something is really wrong, just stop everything
		panic("can't even decode rescue gif")
	}
}

func (g *AnsiGif) Insert(id string) (err error) {
	stmt, err := db.Prepare("INSERT INTO gif (api_id) VALUES (?)")
	defer stmt.Close()
	if err != nil {
		return err
	}
	_, err = stmt.Exec(id)
	if err != nil {
		return err
	}

	for i, srcImg := range g.Rendered {
		stmt, err := db.Prepare("INSERT INTO gif_data (frame_nb, delay, frame, gif_id) VALUES (?, ?, ?, ?)")
		defer stmt.Close()
		if err != nil {
			return err
		}
		_, err = stmt.Exec(i, srcImg.Delay, srcImg.Output, id)
		if err != nil {
			return err
		}
	}
	return nil
}

func (g *AnsiGif) Reverse() {
	for i := len(g.Rendered)/2 - 1; i >= 0; i-- {
		opp := len(g.Rendered) - 1 - i
		g.Rendered[i], g.Rendered[opp] = g.Rendered[opp], g.Rendered[i]
	}
}

func (g *AnsiGif) Preview() (res string) {
	// set image scale factor for ANSIPixel grid, background color and scale mode
	tx, ty := 30, 9
	sfy, sfx := ansimage.BlockSizeY, ansimage.BlockSizeX
	mc := color.RGBA{0x00, 0x00, 0x00, 0xff}
	dm := ansimage.DitheringMode(0)
	sm := ansimage.ScaleMode(2)
	pix, _ := ansimage.NewScaledFromImage(g.Gif.Image[0], sfy*ty, sfx*tx, mc, sm, dm)
	pix.SetMaxProcs(runtime.NumCPU())
	res = pix.Render()
	return
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
