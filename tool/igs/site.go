package igs

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

const (
	httpUserAgent  = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/12.0.3 Safari/605.1.15"
	urlSiteListIGS = "https://network.igs.org/api/public/stations/?include_former=on&format=json&length=2147483647"
)

type SiteInfo struct {
	Name      string     `json:"name"`
	Status    int8       `json:"status"`
	XYZ       [3]float64 `json:"xyz"`
	LLH       [3]float64 `json:"llh"`
	System    []string   `json:"satellite_system"`
	RtSystem  []string   `json:"real_time_systems"`
	AntType   string     `json:"antenna_type"`
	AntSerial string     `json:"antenna_serial_number"`
	AntUNE    [3]float64 `json:"antenna_marker_une"`
	RcvType   string     `json:"receiver_type"`
	RcvSerial string     `json:"serial_number"`
	RcvFirm   string     `json:"firmware"`
	FreqStd   string     `json:"frequency_standard"`
	LastData  string     `json:"last_data_time"`
	JoinDate  string     `json:"join_date"`
	LastPub   string     `json:"last_publish"`
}

type SiteInfoArray struct {
	Num    int64      `json:"recordsFiltered"`
	NumAll int64      `json:"recordsTotal"`
	Array  []SiteInfo `json:"data"`
}

func GetSiteInfoJson(f string) error {
	var jSArray SiteInfoArray
	fp, err := os.OpenFile(f, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0664)

	if err != nil {
		return err
	}

	defer fp.Close()

	client := http.Client{}
	ctx, cancel := context.WithCancel(context.TODO())
	timer := time.AfterFunc(time.Minute, func() { cancel() })
	request, _ := http.NewRequest(http.MethodGet, urlSiteListIGS, nil)
	request.Header.Add("User-Agent", httpUserAgent)
	request = request.WithContext(ctx)

	var response *http.Response

	for i := 0; i < 5; i++ {
		timer.Reset(time.Minute)
		response, err = client.Do(request)

		if err == nil && response.StatusCode == http.StatusOK {
			break
		}
	}

	if err != nil {
		return err
	}

	defer response.Body.Close()

	dcr := json.NewDecoder(response.Body)

	for dcr.More() {
		err = dcr.Decode(&jSArray)

		if err != nil {
			return fmt.Errorf("fialed to parse the response body, %s", err)
		}
	}

	ecr := json.NewEncoder(fp)
	ecr.SetIndent("", "    ")
	ecr.Encode(&jSArray)

	return nil
}
