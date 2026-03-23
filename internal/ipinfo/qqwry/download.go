package qqwry

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	log "unknwon.dev/clog/v2"
)

const (
	DefaultQQWryURL = "https://github.com/metowolf/qqwry.dat/releases/latest/download/qqwry.dat"
)

func resolveDownloadURL(url string) string {
	if strings.TrimSpace(url) == "" {
		return DefaultQQWryURL
	}
	return url
}

func download(url string) (err error) {
	log.Info("Downloading qqwry.dat...")

	client := http.Client{
		Timeout: 90 * time.Second,
	}

	request, err := http.NewRequest(http.MethodGet, resolveDownloadURL(url), nil)
	if err != nil {
		return err
	}
	request.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Warn("%v", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download qqwry.dat, status code: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if len(data) < 1000 {
		return errors.New("downloaded qqwry.dat is too small, possibly corrupt")
	}

	return os.WriteFile(qqwryDataPath, data, 0644)
}
