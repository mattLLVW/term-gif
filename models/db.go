package models

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"log"
)

var db *sql.DB

// Initialise database globally from here
func InitDB(dataSourceName string) {
	var err error
	db, err = sql.Open("mysql", dataSourceName)
	if err != nil {
		log.Panic(err)
	}

	if err = db.Ping(); err != nil {
		log.Panic(err)
	}
}

// Check if id already exist in database
func AlreadyExist(id string) (exist bool) {
	err := db.QueryRow("SELECT IF(COUNT(*),'true','false') FROM gif WHERE tenor_id=?", id).Scan(&exist)
	if err != nil {
		log.Println("can't check if already exist: ", err)
		return false
	}
	return exist
}

// Retrieve all images of a gif sorted by frame_nb by default
func GetGifFromDb(id string, rev bool) (imgs []RenderedImg, err error) {
	order := "ASC"
	if rev {
		order = "DESC"
	}
	rows, err := db.Query(fmt.Sprintf("SELECT delay, frame FROM gif_data WHERE gif_id =? ORDER BY frame_nb %s", order), id)
	if err != nil {
		return imgs, err
	}
	for rows.Next() {
		var img RenderedImg
		err = rows.Scan(&img.Delay, &img.Output)
		if err != nil {
			return imgs, err
		}
		imgs = append(imgs, img)
	}
	return imgs, err
}
