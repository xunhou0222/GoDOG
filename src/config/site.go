package config

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
)

type SiteInfo struct {
	Name string  `json:"name"`
}

type SiteInfoArray struct {
	Array []SiteInfo `json:"data"`
}

func (a SiteInfoArray) Contains(name string) bool {
	name = strings.ToUpper(name)
	return slices.ContainsFunc(a.Array, func(e SiteInfo) bool { return strings.Contains(e.Name, name) })
}

func (a SiteInfoArray) Index(name string) int {
	name = strings.ToUpper(name)
	return slices.IndexFunc(a.Array, func(e SiteInfo) bool { return strings.Contains(e.Name, name) })
}

func parseJsonSiteInfo (f string, jSArray *SiteInfoArray) error {
	fp, err := os.Open(f)

	if err != nil {
		return fmt.Errorf("error occurs while parsing the json file, %s", err)
	}

	defer fp.Close()

	dcr := json.NewDecoder(fp)

	for dcr.More() {
		err = dcr.Decode(&jSArray)

		if err != nil {
			return fmt.Errorf("error occurs while parsing the json file, %s", err)
		}
	}

	if len(jSArray.Array) == 0 {
		return fmt.Errorf("the json file is empty")
	}

	return nil
}