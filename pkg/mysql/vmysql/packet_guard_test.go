package vmysql

import (
	"testing"

	querypb "vitess.io/vitess/go/vt/proto/query"
)

func TestIsEOFPacketEmpty(t *testing.T) {
	if isEOFPacket([]byte{}) {
		t.Fatalf("empty packet should not be EOF packet")
	}
}

func TestIsErrorPacketEmpty(t *testing.T) {
	if isErrorPacket([]byte{}) {
		t.Fatalf("empty packet should not be error packet")
	}
}

func TestParseEOFPacketTooShort(t *testing.T) {
	_, _, err := parseEOFPacket([]byte{EOFPacket})
	if err == nil {
		t.Fatalf("short EOF packet should return error")
	}
}

func TestParseRowOutOfRange(t *testing.T) {
	c := &Conn{}
	fields := []*querypb.Field{
		{Type: 253},
	}
	_, err := c.parseRow([]byte{}, fields)
	if err == nil {
		t.Fatalf("empty row should return out of range error")
	}
}
