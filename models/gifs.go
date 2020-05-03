package models

import (
	"github.com/eliukblau/pixterm/pkg/ansimage"
	"image"
	"image/color"
	"image/gif"
	"image/draw"
	"log"
	"net/http"
	"os"
	"runtime"
)

type RenderedImg struct {
	Output string
	Delay  int
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func AlreadyExist(id string) (exist bool) {
	err := db.QueryRow("SELECT IF(COUNT(*),'true','false') FROM gif WHERE tenor_id=?", id).Scan(&exist)
	if err != nil {
		log.Println("can't check if already exist: ", err)
		return false
	}
	return exist
}

func InsertGif(id string, url string) (imgs []RenderedImg) {
	res, err := http.Get(url)
	// TODO: err
	defer res.Body.Close()
	g, err := gif.DecodeAll(res.Body)
	// TODO: err

	stmt, err := db.Prepare("INSERT INTO gif (tenor_id) VALUES (?)")
	if err != nil {
		log.Println(err)
	}
	_, err = stmt.Exec(id)
	checkErr(err)

	imgs = renderGif(g)
	for i, srcImg := range imgs {
		stmt, err := db.Prepare("INSERT INTO gif_data (frame_nb, delay, frame, gif_id) VALUES (?, ?, ?, ?)")
		if err != nil {
			log.Println(err)
		}
		_, err = stmt.Exec(i, srcImg.Delay, srcImg.Output, id)
		checkErr(err)
	}
	return imgs
}

func GetGifFromDb(id string) (imgs []RenderedImg) {
	rows, err := db.Query("SELECT delay, frame FROM gif_data WHERE gif_id =? ORDER BY frame_nb", id)
	if err != nil {
		log.Println(err)
	}
	for rows.Next() {
		var img RenderedImg
		err = rows.Scan(&img.Delay, &img.Output)
		imgs = append(imgs, img)
	}
	return imgs
}

// Split gif, transform to ansi code and return a slice of images:delay
func renderGif(g *gif.GIF) (imgs []RenderedImg) {
	// https://stackoverflow.com/a/33296596/8135079
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Error while searching gif %s", r)
		}
	}()

	imgWidth, imgHeight := getGifDimensions(g)

	overpaintImage := image.NewRGBA(image.Rect(0, 0, imgWidth, imgHeight))
	draw.Draw(overpaintImage, overpaintImage.Bounds(), g.Image[0], image.ZP, draw.Src)

	// set image scale factor for ANSIPixel grid
	tx, ty := 30, 9
	sfy, sfx := ansimage.BlockSizeY, ansimage.BlockSizeX
	mc := color.RGBA{0x00, 0x00, 0x00, 0xff}
	dm := ansimage.DitheringMode(0)
	sm := ansimage.ScaleMode(2)
	// Clear terminal and position cursor

	for i, srcImg := range g.Image {
		delay := g.Delay[i]
		draw.Draw(overpaintImage, overpaintImage.Bounds(), srcImg, image.ZP, draw.Over)
		pix, _ := ansimage.NewScaledFromImage(overpaintImage, sfy*ty, sfx*tx, mc, sm, dm)
		pix.SetMaxProcs(runtime.NumCPU())
		renderedGif := pix.Render()
		imgs = append(imgs, RenderedImg{Delay: delay, Output: renderedGif})
	}
	return imgs
}

// Get max Gif dimensions.
func getGifDimensions(gif *gif.GIF) (x, y int) {
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
func OopsGif() ([]RenderedImg) {
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
