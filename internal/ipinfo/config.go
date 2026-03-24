package ipinfo

type Config struct {
	Database      string `yaml:"database"`
	GeoLicenseKey string `yaml:"geo_license_key"`
	QQwryURL      string `yaml:"qqwry_url"`
	GeoIPURL      string `yaml:"geoip_url"`
}
