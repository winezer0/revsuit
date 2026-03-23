package qqwry

import (
	"os"
	"sync"

	"github.com/sinlov/qqwry-golang/qqwry"
	log "unknwon.dev/clog/v2"
)

type Database struct {
	wry *qqwry.QQwry
}

// Area returns IpArea according to ip
func (db *Database) Area(ip string) string {
	defer func() {
		_ = recover()
	}()
	if db.wry == nil {
		return ""
	}
	ipData := db.wry.SearchByIPv4(ip)
	if ipData.Area == " CZ88.NET" {
		return ipData.Country
	}
	return ipData.Country + " " + ipData.Area
}

var wry *qqwry.QQwry
var once sync.Once
var qqwryDataPath = "qqwry.dat"

func initFromFile() bool {
	qqwry.DatData.FilePath = qqwryDataPath
	init := qqwry.DatData.InitDatFile()
	if v, ok := init.(error); ok {
		if v != nil {
			return false
		}
	}
	return true
}

func New(downloadURL string) *Database {
	once.Do(func() {
		if _, err := os.Stat(qqwryDataPath); err == nil {
			if !initFromFile() {
				log.Warn("qqwry file init failed")
				wry = nil
				return
			}
		} else if os.IsNotExist(err) {
			err := download(downloadURL)
			if err != nil {
				log.Warn("Download qqwry.dat failed, caused by:%v, recommend to download it by yourself otherwise the `IpArea` will be null", err)
				wry = nil
				return
			}
			if !initFromFile() {
				log.Warn("qqwry file init failed after download")
				wry = nil
				return
			}
		} else {
			log.Warn("Stat qqwry.dat failed, caused by:%v", err)
			wry = nil
			return
		}
		wry = qqwry.NewQQwry()
	})
	return &Database{wry: wry}
}
