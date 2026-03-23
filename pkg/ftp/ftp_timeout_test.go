package ftp

import (
	"testing"
	"time"
)

func TestWaitUploadDataSuccess(t *testing.T) {
	ch := make(chan []byte, 1)
	ch <- []byte("ok")
	data, ok := waitUploadData(ch, time.Second)
	if !ok {
		t.Fatalf("expected upload data")
	}
	if string(data) != "ok" {
		t.Fatalf("expected ok, got %s", string(data))
	}
}

func TestWaitUploadDataTimeout(t *testing.T) {
	ch := make(chan []byte)
	_, ok := waitUploadData(ch, 20*time.Millisecond)
	if ok {
		t.Fatalf("expected timeout")
	}
}
