package rmi

import "testing"

func TestParseRMIPathSuccess(t *testing.T) {
	data := []byte{1, 2, 3, 0xdf, 0x74, 0x01, 0x02, 'x', 'y', 'z', 0x00}
	path, ok := parseRMIPath(data, len(data))
	if !ok {
		t.Fatalf("expected parse success")
	}
	if path != "xyz" {
		t.Fatalf("expected xyz, got %s", path)
	}
}

func TestParseRMIPathInvalid(t *testing.T) {
	data := []byte{0xdf, 0x74, 0x01}
	_, ok := parseRMIPath(data, len(data))
	if ok {
		t.Fatalf("expected parse failure")
	}
}
