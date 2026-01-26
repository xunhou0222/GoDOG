package main

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

/***** METHOD **********************************/

func (a TargetInfoArray) Contains(name string) bool {
	name = strings.ToUpper(name)

	caller := func(e TargetInfo) bool {
		return strings.Contains(e.Name, name)
	}

	return slices.ContainsFunc(a.Array, caller)
}

/***********************************************/

func (a TargetInfoArray) Index(name string) int {
	name = strings.ToUpper(name)

	caller := func(e TargetInfo) bool {
		return strings.Contains(e.Name, name)
	}

	return slices.IndexFunc(a.Array, caller)
}

/***********************************************/

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
		return fmt.Errorf("empty file")
	}

	return nil
}

/***********************************************/
