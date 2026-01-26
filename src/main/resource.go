package main

import (
	"encoding/json"
	"fmt"
	"godog/datetime"
	"godog/network"
	"os"
)

/***** STRUCT **********************************/

type tResource struct {
	Sources  []network.NetworkInfo `json:"sources"`
	TimeSys  string                `json:"time system"`
	Interval int                   `json:"interval"`
}

/***********************************************/

type Resource struct {
	Sources  []network.NetworkInfo
	TimeSys  datetime.TimeSys
	Interval int
}

/***** FUNCTION ********************************/

func ParseResourceJson(rsFile string, rsMap map[string]Resource) error {
	fp, err := os.Open(rsFile)

	if err != nil {
		return err
	}

	defer fp.Close()

	dcr := json.NewDecoder(fp)
	jTmp := make(map[string]tResource)

	for dcr.More() {
		err = dcr.Decode(&jTmp)

		if err != nil {
			return err
		}
	}

	for kw, val := range jTmp {
		var rs Resource

		rs.TimeSys = datetime.ParseTimeSys(val.TimeSys)

		if val.Interval <= 0 {
			return fmt.Errorf(`invalid "interval" of resource "%s"`, kw)
		}

		rs.Interval = val.Interval

		for _, s := range val.Sources {
			if !(s.IsFtp() || s.IsFtps() || s.IsHttp() || s.IsHttps() || s.IsHttpsCddis()) {
				return fmt.Errorf(`unsupported url type for resource "%s"`, kw)
			}

			if s.UserName == "" {
				s.UserName = "anonymous"
			}

			if s.Password == "" {
				s.Password = "anonymous"
			}

			rs.Sources = append(rs.Sources, s)
		}

		rsMap[kw] = rs
	}

	return nil
}

/***********************************************/
