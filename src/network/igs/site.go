package igs

import (
	"slices"
	"strings"
)

type tmpRECEIVER struct {
	Name       string `json:"Name"`
	System     string `json:"SatelliteSystem"`
	Serial     string `json:"SerialNumber"`
	Firmware   string `json:"FirmwareVersion"`
	ElevCutoff string `json:"ElevCutoff"`
	DateSince  string `json:"DateInstalled"`
}

type RECEIVER struct {
	Name       string
	System     string
	Serial     string
	Firmware   string
	ElevCutoff float32
	DateSince  string
}

type tmpANTENNA struct {
	Name      string `json:"Name"`
	Radome    string `json:"Radome"`
	Serial    string `json:"SerialNumber"`
	Arp       string `json:"ARP"`
	Up        string `json:"MarkerUp"`
	North     string `json:"MarkerNorth"`
	East      string `json:"MarkerEast"`
	DateSince string `json:"DateInstalled"`
}

type ANTENNA struct {
	Name      string
	Radome    string
	Serial    string
	Arp       string
	Up        float64
	North     float64
	East      float64
	DateSince string
}


type tmpCLOCK struct {
	Type      string `json:"Type"`
	InputFreq string `json:"InputFrequency"`
	DateSince string `json:"EffectiveDates"`
}

type CLOCK struct {
	Type      string  `json:"Type"`
	InputFreq float64 `json:"InputFrequency"`
	DateSince string  `json:"EffectiveDates"`
}

type tmpSiteInfo struct {
	Name      string      `json:"Name"`
	X         float64     `json:"X"`
	Y         float64     `json:"Y"`
	Z         float64     `json:"Z"`
	Latitude  string      `json:"Latitude"`
	Longitude string      `json:"Longitude"`
	Height    string      `json:"Height"`
	Receiver  tmpRECEIVER `json:"Receiver"`
	Antenna   tmpANTENNA  `json:"Antenna"`
	Clock     tmpCLOCK    `json:"Clock"`
}

type SiteInfo struct {
	Name      string
	X         float64
	Y         float64
	Z         float64
	Latitude  float64
	Longitude float64
	Height    float64
	Receiver  tmpRECEIVER
	Antenna   tmpANTENNA
	Clock     tmpCLOCK 
}

type tmpSiteInfoArray struct {
	Array []tmpSiteInfo `json:"Sites"`
}

type SiteInfoArray struct {
	Array []SiteInfo
}

func (a *SiteInfoArray) Contains(name string) bool {
	name = strings.ToUpper(name)
	return slices.ContainsFunc(a.Array, func(e SiteInfo) bool { return strings.Contains(e.Name, name) })
}

func (a *SiteInfoArray) Index(name string) int {
	name = strings.ToUpper(name)
	return slices.IndexFunc(a.Array, func(e SiteInfo) bool { return strings.Contains(e.Name, name) })
}
