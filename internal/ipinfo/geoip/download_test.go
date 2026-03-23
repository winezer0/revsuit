package geoip

import "testing"

func TestResolveDownloadURLWithDefault(t *testing.T) {
	url := resolveDownloadURL("abc", "")
	expected := "https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-City&license_key=abc&suffix=tar.gz"
	if url != expected {
		t.Fatalf("unexpected default url: %s", url)
	}
}

func TestResolveDownloadURLWithTemplate(t *testing.T) {
	url := resolveDownloadURL("def", "https://example.com/geoip?license=%s")
	if url != "https://example.com/geoip?license=def" {
		t.Fatalf("unexpected template url: %s", url)
	}
}

func TestResolveDownloadURLWithFixedURL(t *testing.T) {
	url := resolveDownloadURL("ghi", "https://example.com/static.mmdb")
	if url != "https://example.com/static.mmdb" {
		t.Fatalf("unexpected fixed url: %s", url)
	}
}
