package config

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
)

/***** STRUCT **********************************/

type TargetInfo struct {
	Name string `json:"name"`
}

/***** STRUCT **********************************/

type TargetInfoArray struct {
	Array []TargetInfo `json:"data"`
}

func (a TargetInfoArray) Contains(name string) bool {
	name = strings.ToUpper(name)
	return slices.ContainsFunc(a.Array, func(e TargetInfo) bool { return strings.Contains(e.Name, name) })
}

func (a TargetInfoArray) Index(name string) int {
	name = strings.ToUpper(name)
	return slices.IndexFunc(a.Array, func(e TargetInfo) bool { return strings.Contains(e.Name, name) })
}

func (a *TargetInfoArray) parseJson(f string) error {
	fp, err := os.Open(f)

	if err != nil {
		return fmt.Errorf("error occurs while parsing the json file, %s", err)
	}

	defer fp.Close()

	dcr := json.NewDecoder(fp)

	for dcr.More() {
		err = dcr.Decode(a)

		if err != nil {
			return fmt.Errorf("error occurs while parsing the json file, %s", err)
		}
	}

	if len(a.Array) == 0 {
		return fmt.Errorf("the json file is empty")
	}

	return nil
}

func (a TargetInfoArray) getArray() []TargetInfo {
	return a.Array
}

/***********************************************/
