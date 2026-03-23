package ldap

import "testing"

func TestParsePathSuccess(t *testing.T) {
	payload := []byte{0, 0, 0, 0, 0, 0, 0, 0, 3, 'a', 'b', 'c'}
	path, ok := parsePath(payload, len(payload))
	if !ok {
		t.Fatalf("expected parse success")
	}
	if path != "abc" {
		t.Fatalf("expected abc, got %s", path)
	}
}

func TestParsePathInvalidLength(t *testing.T) {
	payload := []byte{0, 0, 0, 0, 0, 0, 0, 0, 5, 'a', 'b'}
	_, ok := parsePath(payload, len(payload))
	if ok {
		t.Fatalf("expected parse failure")
	}
}
