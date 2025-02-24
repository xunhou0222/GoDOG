package config

import "strings"

type Task struct {
	Type     string
	Path     string
	Backward int
	Forward  int
	IfUnzip  bool
}

type tmpTask struct {
	Type     string   `json:"type"`
	Path     string   `json:"path"`
	Backward int      `json:"backward"`
	Forward  int      `json:"forward"`
	IfUnzip  string   `json:"uncompress"`
	InfoFile string   `json:"information"`
	Targets  []string `json:"targets"`
}

func (t *Task) IsRnxIGSTask() bool {
	return strings.Contains(strings.ToLower(t.Type), "rnx") &&
		strings.Contains(strings.ToUpper(t.Type), "IGS")
}

func (t *Task) IsCrdILRSTask() bool {
	return strings.Contains(strings.ToLower(t.Type), "crd") &&
		strings.Contains(strings.ToUpper(t.Type), "ILRS")
}
