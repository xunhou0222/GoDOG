package igs

import (
	"encoding/csv"
	"fmt"
	"godog/network"
	"io"
	"os"
	"strings"
)

const (
	urlCsvIGS = "https://files.igs.org/pub/station/general/IGSNetwork.csv?_gl=1*v6se9u*_ga*MTUzMjY0Nzc2MS4xNzI1ODY0MTc0*_ga_Z5RH7R682C*MTcyNTg4MjY4NC4yLjEuMTcyNTg4Mjc4OS41NC4wLjA.&_ga=2.176661605.196205060.1725864174-1532647761.1725864174"
)

func GetCsvIGS(path string) network.TaskError {
	// download
	f := &network.NetTask{Source: network.NetInfo{URL: urlCsvIGS}, 
                          Path: path,
						  Size: 0, 
						  Continue: false}
	terr := network.HTTPDownload(f)

	if terr != nil {
		err := fmt.Errorf("error occurs while downloading the json file, %s", terr)
		return network.NewTaskError(err, terr.Temporary())
	}

	return nil
}

func ParseCsvIGS(path string, cSiteArray *SiteInfoArray) error {
	fp, err := os.Open(path)

	if err != nil {
		err = fmt.Errorf("error occurs while parsing the csv file, %s", err)
		return err
	}

	defer fp.Close()

	rdr := csv.NewReader(fp)

	var record []string
	var cSta SiteInfo

	var idxName, idxX, idxY, idxZ, idxLat, idxLong, idxH int
	var idxRcvName, idxRcvSys, idxRcvSeri, idxRcvFirm, idxRcvElev, idxRcvDate int
	var idxAntName, idxAntRadome, idxAntSeri, idxAntArp, idxAntU, idxAntN, idxAntE, idxAntDate int
	var idxClkType, idxClkFreq, idxClkDate int

	for i := 0; ; i++ {
		if record, err = rdr.Read(); err != nil {
			break
		}

		if i == 0 {
			for idx, val := range record {
				if strings.Contains(val, "StationName") {
					idxName = idx
				} else if strings.Contains(val, "X") {
					idxX = idx
				} else if strings.Contains(val, "Y") {
					idxY = idx
				} else if strings.Contains(val, "Z") {
					idxZ = idx
				} else if strings.Contains(val, "Latitude") {
					idxLat = idx
				} else if strings.Contains(val, "Longitude") {
					idxLong = idx
				} else if strings.Contains(val, "Height") {
					idxH = idx
				} else if strings.Contains(val, "ReceiverName") {
					idxRcvName = idx
				} else if strings.Contains(val, "ReceiverSatelliteSystem") {
					idxRcvSys = idx
				} else if strings.Contains(val, "ReceiverSerialNumber") {
					idxRcvSeri = idx
				} else if strings.Contains(val, "ReceiverFirmwareVersion") {
					idxRcvFirm = idx
				} else if strings.Contains(val, "ReceiverElevationCutoff") {
					idxRcvElev = idx
				} else if strings.Contains(val, "ReceiverDateInstalled") {
					idxRcvDate = idx
				} else if strings.Contains(val, "AntennaName") {
					idxAntName = idx
				} else if strings.Contains(val, "AntennaRadome") {
					idxAntRadome = idx
				} else if strings.Contains(val, "AntennaSerialNumber") {
					idxAntSeri = idx
				} else if strings.Contains(val, "AntennaARP") {
					idxAntArp = idx
				} else if strings.Contains(val, "AntennaMarkerUp") {
					idxAntU = idx
				} else if strings.Contains(val, "AntennaMarkerNorth") {
					idxAntN = idx
				} else if strings.Contains(val, "AntennaMarkerEast") {
					idxAntE = idx
				} else if strings.Contains(val, "AntennaDateInstalled") {
					idxAntDate = idx
				} else if strings.Contains(val, "ClockType") {
					idxClkType = idx
				} else if strings.Contains(val, "ClockInputFrequency") {
					idxClkFreq = idx
				} else if strings.Contains(val, "ClockEffectiveDates") {
					idxClkDate = idx
				}
			}

			continue
		}

		cSta.Name = record[idxName]
		fmt.Sscanf(record[idxX], "%f", &cSta.X)
		fmt.Sscanf(record[idxY], "%f", &cSta.Y)
		fmt.Sscanf(record[idxZ], "%f", &cSta.Z)
		fmt.Sscanf(record[idxLat], "%f", &cSta.Latitude)
		fmt.Sscanf(record[idxLong], "%f", &cSta.Longitude)
		fmt.Sscanf(record[idxH], "%f", &cSta.Height)

		cSta.Receiver.Name = record[idxRcvName]
		cSta.Receiver.System = record[idxRcvSys]
		cSta.Receiver.Serial = record[idxRcvSeri]
		cSta.Receiver.Firmware = record[idxRcvFirm]
		fmt.Sscanf(record[idxRcvElev], "%f", &cSta.Receiver.ElevCutoff)
		cSta.Receiver.DateSince = record[idxRcvDate]

		cSta.Antenna.Name = record[idxAntName]
		cSta.Antenna.Radome = record[idxAntRadome]
		cSta.Antenna.Serial = record[idxAntSeri]
		cSta.Antenna.Arp = record[idxAntArp]
		fmt.Sscanf(record[idxAntU], "%f", &cSta.Antenna.Up)
		fmt.Sscanf(record[idxAntN], "%f", &cSta.Antenna.North)
		fmt.Sscanf(record[idxAntE], "%f", &cSta.Antenna.East)
		cSta.Antenna.DateSince = record[idxAntDate]

		cSta.Clock.Type = record[idxClkType]
		fmt.Sscanf(record[idxClkFreq], "%f", &cSta.Clock.InputFreq)
		cSta.Clock.DateSince = record[idxClkDate]

		cSiteArray.Array = append(cSiteArray.Array, cSta)
	}

	if err != io.EOF {
		err = fmt.Errorf("error occurs while parsing the csv file, %s", err)
		return err
	}

	return nil
}
