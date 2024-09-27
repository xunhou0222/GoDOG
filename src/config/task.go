package config

import "strings"

type Task struct {
	Type     string `json:"type"`
	Path     string `json:"path"`
	Backward int    `json:"backward"`
	Forward  int    `json:"forward"`
}

func (t *Task) IsRnxIGSTask() bool {
	return strings.Contains(strings.ToLower(t.Type), "rnx") && 
	       strings.Contains(strings.ToUpper(t.Type), "IGS")
}
