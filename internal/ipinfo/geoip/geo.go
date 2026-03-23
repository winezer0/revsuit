package geoip

import (
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/oschwald/geoip2-golang"
	log "unknwon.dev/clog/v2"
)

type Database struct {
	geo *geoip2.Reader
}

var geo *geoip2.Reader
var once sync.Once
var geoipDataPath = "GeoLite2-City.mmdb"

// Area returns IpArea according to ip
func (db *Database) Area(ip string) string {
	defer func() {
		_ = recover()
	}()
	record, err := db.geo.City(net.ParseIP(ip))
	if err != nil {
		return ""
	}

	country := record.Country.Names["en"]
	city := record.City.Names["en"]
	if city == "" {
		city = record.Location.TimeZone
	}
	return fmt.Sprintf("%s %s", country, city)
}

func New(licenseKey, downloadURL string) *Database {
	once.Do(func() {
		var err error
		if _, statErr := os.Stat(geoipDataPath); os.IsNotExist(statErr) {
			err = download(licenseKey, downloadURL)
			if err != nil {
				log.Warn("Download GeoLite2-City.mmdb failed, caused by:%v, recommend to download it by yourself otherwise the `IpArea` will be null", err)
			}
		} else if statErr != nil {
			log.Warn("Stat GeoLite2-City.mmdb failed, caused by:%v", statErr)
		}
		geo, err = geoip2.Open(geoipDataPath)
		if err != nil {
			log.Error("Load GeoLite2-City.mmdb failed, `IpArea` will be null")
		}
	})
	return &Database{geo: geo}
}
