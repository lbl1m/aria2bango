// Package peerid provides peer ID parsing functionality
package peerid

import (
	"net/url"
	"strings"
)

// ClientInfo contains parsed client information from peer ID
type ClientInfo struct {
	Name    string
	Version string
}

// azureusClients maps Azureus-style peer ID prefixes (2 chars after -) to client names
// Format: -XXYYYY- where XX is client code, YYYY is version
var azureusClients = map[string]string{
	"BC": "BitComet",
	"qB": "qBittorrent",
	"UT": "uTorrent",
	"TR": "Transmission",
	"DE": "Deluge",
	"AZ": "Azureus/Vuze",
	"XL": "Xunlei",
	"SD": "Xunlei",
	"BN": "Baidu Netdisk",
	"HP": "HP Play",
	"MG": "MediaGet",
	"AR": "Ares",
	"AT": "Artemis",
	"AV": "Avicora",
	"AG": "Ares",
	"BB": "BitBuddy",
	"BE": "BitTorrent SDK",
	"BG": "BTG",
	"BH": "BitZilla",
	"BM": "BitMagnet",
	"BP": "BitTorrent Pro",
	"BR": "BitRocket",
	"BS": "BTSlave",
	"BT": "BitTorrent",
	"BW": "BitWombat",
	"BX": "Bittorrent X",
	"CD": "Enhanced CTorrent",
	"CT": "CTorrent",
	"DP": "Propagate Data Client",
	"EB": "EBit",
	"FC": "FileCroc",
	"FD": "Free Download Manager",
	"FG": "FlashGet",
	"FL": "Flud",
	"FT": "FoxTorrent",
	"FW": "FrostWire",
	"FX": "Freebox",
	"GS": "GSTorrent",
	"HK": "Hekate",
	"HL": "Halite",
	"HN": "Hydranode",
	"KG": "KGet",
	"KT": "KTorrent",
	"LC": "LeechCraft",
	"LH": "LH-ABC",
	"LP": "Lphant",
	"LT": "libtorrent",
	"LW": "LimeWire",
	"Lr": "LibTorrent (Rasterbar)",
	"MK": "Meerkat",
	"ML": "MLDonkey",
	"MO": "MonoTorrent",
	"MP": "MooPolice",
	"MR": "Miro",
	"MT": "Moonlight Torrent",
	"NB": "Net::BitTorrent",
	"NX": "Net Transport",
	"OS": "OneSwarm",
	"OT": "OmegaTorrent",
	"PD": "Pando",
	"PI": "PicoTorrent",
	"QD": "QQDownload",
	"QT": "Qt 4 Torrent example",
	"RS": "Rufus",
	"RT": "Retriever",
	"RZ": "RezTorrent",
	"SB": "Swiftbit",
	"SM": "SoMud",
	"SN": "ShareNet",
	"SP": "BitSpirit",
	"SS": "SwarmScope",
	"ST": "SymTorrent",
	"SZ": "Shareaza",
	"TB": "Torch Browser",
	"TE": "Tribler",
	"TL": "Tribler",
	"TN": "TorrentDotNET",
	"TS": "TorrentStorm",
	"TT": "TuoTu",
	"UL": "uLeecher",
	"UM": "uTorrent Mac",
	"UW": "uTorrent Web",
	"VG": "Vagaa",
	"WD": "WebTorrent Desktop",
	"WT": "BitLet",
	"WW": "WebTorrent",
	"WY": "FireTorrent",
	"XC": "Xtorrent",
	"XF": "Xfplay",
	"XT": "Xtorrent",
	"XX": "Xtorrent",
	"XY": "Xunlei",
	"XZ": "Xunlei",
	"ZT": "ZipTorrent",
	"ZP": "ZipTorrent",
	"ZZ": "ZipTorrent",
}

// shadowClients maps Shadow's-style peer ID prefixes (single char) to client names
var shadowClients = map[string]string{
	"M": "BitTorrent Mainline",
	"A": "ABC",
	"O": "Osprey",
	"Q": "BTQueue",
	"R": "Tribler",
	"S": "Shad0w",
	"T": "BitTornado",
	"U": "UPnP NAT Bit Torrent",
}

// Parse parses a peer ID and returns client information
func Parse(peerID string) ClientInfo {
	// Handle URL-encoded peer IDs
	if strings.Contains(peerID, "%") {
		if decoded, err := url.QueryUnescape(peerID); err == nil {
			peerID = decoded
		}
	}

	// Check Azureus-style peer IDs: -XXYYYY-
	if strings.HasPrefix(peerID, "-") && len(peerID) >= 8 {
		// Extract the 2-character client code
		clientCode := peerID[1:3]
		if name, ok := azureusClients[clientCode]; ok {
			info := ClientInfo{
				Name: name,
			}
			// Extract version (4 characters after client code)
			if len(peerID) >= 7 {
				versionRaw := peerID[3:7]
				info.Version = formatVersion(versionRaw)
			}
			return info
		}
	}

	// Check Shadow's style peer IDs (single character prefix)
	if len(peerID) >= 1 {
		prefix := string(peerID[0])
		if name, ok := shadowClients[prefix]; ok {
			return ClientInfo{
				Name:    name,
				Version: "",
			}
		}
	}

	// Check for mainline style: MYYYP...
	if len(peerID) >= 4 && peerID[0] == 'M' {
		return ClientInfo{
			Name:    "BitTorrent Mainline",
			Version: formatVersion(peerID[1:4]),
		}
	}

	return ClientInfo{
		Name:    "Unknown",
		Version: "",
	}
}

// formatVersion formats a raw version string into a readable version
func formatVersion(raw string) string {
	if len(raw) < 4 {
		return raw
	}

	// Handle version formats like "0213" -> "2.13" or "5120" -> "51.20"
	// Common format: XXYY where XX is major, YY is minor
	major := raw[:2]
	minor := raw[2:4]

	// Parse major version
	majorNum := 0
	for _, c := range major {
		if c >= '0' && c <= '9' {
			majorNum = majorNum*10 + int(c-'0')
		}
	}

	// Parse minor version
	minorNum := 0
	for _, c := range minor {
		if c >= '0' && c <= '9' {
			minorNum = minorNum*10 + int(c-'0')
		}
	}

	// Format version
	if majorNum > 0 {
		if minorNum > 0 {
			return strings.TrimLeft(major, "0") + "." + strings.TrimLeft(minor, "0")
		}
		return strings.TrimLeft(major, "0")
	}

	return raw
}

// GetName returns the client name from a peer ID
func GetName(peerID string) string {
	return Parse(peerID).Name
}

// GetNameWithVersion returns the client name with version from a peer ID
func GetNameWithVersion(peerID string) string {
	info := Parse(peerID)
	if info.Version != "" {
		return info.Name + " " + info.Version
	}
	return info.Name
}
