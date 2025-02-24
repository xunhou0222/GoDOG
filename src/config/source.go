package config

import (
	"encoding/json"
	"fmt"
	"godog/gnsstime"
	"godog/network"
	"os"
)

type tmpNetSource struct {
	Sources  []network.NetInfo `json:"sources"`
	TimeSys  string            `json:"time system"`
	Interval int               `json:"interval"`
}

type NetSource struct {
	Sources  []network.NetInfo
	TimeSys  byte
	Interval int
}

func parseJsonSource(path string, jNetSource map[string]NetSource) error {
	fp, err := os.Open(path)

	if err != nil {
		return fmt.Errorf("error occurs while parsing the json file, %s", err)
	}

	defer fp.Close()

	dcr := json.NewDecoder(fp)
	jTmp := make(map[string]tmpNetSource)

	for dcr.More() {
		err = dcr.Decode(&jTmp)

		if err != nil {
			return fmt.Errorf("error occurs while parsing the json file, %s", err)
		}
	}

	for kw, val := range jTmp {
		var ns NetSource

		ns.TimeSys = gnsstime.ParseTimeSys(val.TimeSys)

		if val.Interval <= 0 {
			return fmt.Errorf(`invalid "interval" for "%s"`, kw)
		}

		ns.Interval = val.Interval

		for _, s := range val.Sources {
			if !(s.IsFtp() || s.IsFtps() || s.IsHttp() || s.IsHttps() || s.IsHttpsCddis()) {
				return fmt.Errorf(`unsupported type of "URL" for "%s"`, kw)
			}

			if s.UserName == "" {
				s.UserName = "anonymous"
			}

			if s.Password == "" {
				s.Password = "anonymous"
			}

			ns.Sources = append(ns.Sources, s)
		}

		jNetSource[kw] = ns
	}

	return nil
}
