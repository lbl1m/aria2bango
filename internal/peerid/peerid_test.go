package peerid

import (
	"testing"
)

func TestParseBitComet(t *testing.T) {
	// URL-encoded peer ID: -BC0213-
	peerID := "%2DBC0213%2D%00H%E7%93%28%0C%2A%5EGr%EA%86"
	info := Parse(peerID)

	if info.Name != "BitComet" {
		t.Errorf("Expected BitComet, got %s", info.Name)
	}
	if info.Version != "2.13" {
		t.Errorf("Expected version 2.13, got %s", info.Version)
	}
}

func TestParseQBitTorrent(t *testing.T) {
	peerID := "-qB5120-ME_GpvJS-s49"
	info := Parse(peerID)

	if info.Name != "qBittorrent" {
		t.Errorf("Expected qBittorrent, got %s", info.Name)
	}
	if info.Version != "51.20" {
		t.Errorf("Expected version 51.20, got %s", info.Version)
	}
}

func TestParseTransmission(t *testing.T) {
	peerID := "-TR2940-xxxxxxxxxxxx"
	info := Parse(peerID)

	if info.Name != "Transmission" {
		t.Errorf("Expected Transmission, got %s", info.Name)
	}
}

func TestParseUTorrent(t *testing.T) {
	peerID := "-UT3456-xxxxxxxxxxxx"
	info := Parse(peerID)

	if info.Name != "uTorrent" {
		t.Errorf("Expected uTorrent, got %s", info.Name)
	}
}

func TestParseDeluge(t *testing.T) {
	peerID := "-DE1234-xxxxxxxxxxxx"
	info := Parse(peerID)

	if info.Name != "Deluge" {
		t.Errorf("Expected Deluge, got %s", info.Name)
	}
}

func TestParseXunlei(t *testing.T) {
	peerID := "-XL0012-xxxxxxxxxxxx"
	info := Parse(peerID)

	if info.Name != "Xunlei" {
		t.Errorf("Expected Xunlei, got %s", info.Name)
	}
}

func TestParseUnknown(t *testing.T) {
	peerID := "unknown-peer-id-format"
	info := Parse(peerID)

	if info.Name != "Unknown" {
		t.Errorf("Expected Unknown, got %s", info.Name)
	}
}

func TestGetNameWithVersion(t *testing.T) {
	tests := []struct {
		peerID   string
		expected string
	}{
		{"-qB5120-ME_GpvJS-s49", "qBittorrent 51.20"},
		{"-BC0213-xxxxxxxxxxxx", "BitComet 2.13"},
		{"-TR2940-xxxxxxxxxxxx", "Transmission 29.40"},
		{"unknown-format", "Unknown"},
	}

	for _, tt := range tests {
		result := GetNameWithVersion(tt.peerID)
		if result != tt.expected {
			t.Errorf("GetNameWithVersion(%s) = %s, expected %s", tt.peerID, result, tt.expected)
		}
	}
}

func TestURLDecodedPeerID(t *testing.T) {
	// Test that URL-encoded peer IDs are properly decoded
	peerID := "%2DqB5120%2DME_GpvJS-s49"
	info := Parse(peerID)

	if info.Name != "qBittorrent" {
		t.Errorf("Expected qBittorrent, got %s", info.Name)
	}
}
