package igs

import (
	"bufio"
	"encoding/json"
	"fmt"
	"godog/network"
	"os"
	"path/filepath"
	"strings"
)

const (
	urlJsonIGS = "https://files.igs.org/pub/station/general/IGSNetworkWithFormer.json?_gl=1*1ru3bv0*_ga*MTMwNTE1NzU3MC4xNzE2NDQzNjEy*_ga_Z5RH7R682C*MTcyNTYwNDAxOS4xMC4xLjE3MjU2MDQxMTQuMjYuMC4w&_ga=2.150104118.1229314701.1725604020-1305157570.1716443612"
)

func GetJsonIGS(path string) network.TaskError {
	// download
	f := &network.NetTask{Source: network.NetInfo{URL: urlJsonIGS}, 
                          Path: path,
						  Size: 0, 
						  Continue: false}
	terr := network.HTTPDownload(f)

	if terr != nil {
		err := fmt.Errorf("error occurs while downloading the json file, %s", terr)
		return network.NewTaskError(err, terr.Temporary())
	}

	// modify the json file to the format that can be parsed using "encoding/json"
	var tmpPath = filepath.Join(filepath.Dir(path), "."+filepath.Base(path))

	fpIn, err := os.Open(path)

	if err != nil {
		err = fmt.Errorf("error occurs while modifying the json file, %s", err)
		return network.NewTaskError(err, false)
	}

	scanner := bufio.NewScanner(fpIn)

	fpOut, err := os.Create(tmpPath)

	if err != nil {
		err = fmt.Errorf("error occurs while modifying the json file, %s", err)
		return network.NewTaskError(err, false)
	}

	writer := bufio.NewWriter(fpOut)

	var strline string

	for scanner.Scan() {
		strline = scanner.Text()

		if strline[0] == '{' {
			strline = strline + "\n    \"Sites\": [\n"
		} else if strline[0] == '}' {
			strline = "    ]\n" + strline + "\n"
		} else if strline[0:4] == "    " && len(strline) >= 8 &&
			strline[4:8] != "    " && strline[4] != '}' { // station name
			idx := strings.IndexByte(strline, ':')
			strline = "        {\n            \"Name\": " + strings.Trim(strline[:idx], " ") + ",\n"
		} else {
			strline = "    " + strline + "\n"
		}

		writer.WriteString(strline)
	}

	writer.Flush()
	fpIn.Close()
	fpOut.Close()

	err = os.Rename(tmpPath, path)

	if err != nil {
		err = fmt.Errorf("error occurs while modifying the json file, %s", err)
		return network.NewTaskError(err, false)
	}

	return nil
}

func ParseJsonIGS(path string, jSiteArray *SiteInfoArray) error {
	fp, err := os.Open(path)

	if err != nil {
		err = fmt.Errorf("error occurs while parsing the json file, %s", err)
		return err
	}

	defer fp.Close()

	dcr := json.NewDecoder(fp)
	var jTmp tmpSiteInfoArray

	for dcr.More() {
		err = dcr.Decode(&jTmp)

		if err != nil {
			err = fmt.Errorf("error occurs while parsing the json file, %s", err)
			return err
		}
	}

	var si SiteInfo

	for _, val := range jTmp.Array {
		si.Name = val.Name
		si.X    = val.X
		si.Y    = val.Y
		si.Z    = val.Z
		fmt.Sscanf(val.Latitude, "%f", &si.Latitude)
		fmt.Sscanf(val.Longitude, "%f", &si.Longitude)
		fmt.Sscanf(val.Height, "%f", &si.Height)

		si.Receiver.Name     = val.Receiver.Name
		si.Receiver.System   = val.Receiver.System
		si.Receiver.Serial   = val.Receiver.Serial
		si.Receiver.Firmware = val.Receiver.Firmware
		fmt.Sscanf(val.Receiver.ElevCutoff, "%f", &si.Receiver.ElevCutoff)
		si.Receiver.DateSince = val.Receiver.DateSince

		si.Antenna.Name   = val.Antenna.Name
		si.Antenna.Radome = val.Antenna.Radome
		si.Antenna.Serial = val.Antenna.Serial
		si.Antenna.Arp    = val.Antenna.Arp
		fmt.Sscanf(val.Antenna.Up, "%f", &si.Antenna.Up)
		fmt.Sscanf(val.Antenna.North, "%f", &si.Antenna.North)
		fmt.Sscanf(val.Antenna.East, "%f", &si.Antenna.East)
		si.Antenna.DateSince = val.Antenna.DateSince

		si.Clock.Type = val.Clock.Type
		fmt.Sscanf(val.Clock.InputFreq, "%f", &si.Clock.InputFreq)
		si.Clock.DateSince = val.Clock.DateSince

		jSiteArray.Array = append(jSiteArray.Array, si)
	}

	return nil
}
