package config

import (
	"encoding/json"
	"fmt"
	"godog/gnsstime"
	"os"
	"path/filepath"
	"strings"
)

const (
	MinGoroutineNum = 1
	MaxGoroutineNum = 200
)

var (
	targetInfoMap map[string]*TargetInfoArray = make(map[string]*TargetInfoArray)
	TargetListMap map[string][]string         = make(map[string][]string)
	NetSourceMap  map[string]NetSource        = make(map[string]NetSource)
)

type Config struct {
	StTime  gnsstime.GNSSTime
	EdTime  gnsstime.GNSSTime
	GoNum   int
	LogFile string
	Tasks   []Task
	JobNum  int
}

type tmpConfig struct {
	StTime     string    `json:"start time"`
	EdTime     string    `json:"end time"`
	GoNum      int       `json:"goroutine num"`
	SourceFile string    `json:"source file"`
	LogFile    string    `json:"log file"`
	Tasks      []tmpTask `json:"tasks"`
}

func (cfg *Config) ParseJson(path string) error {
	fp, err := os.Open(path)

	if err != nil {
		return err
	}

	defer fp.Close()

	dcr := json.NewDecoder(fp)
	var cfgTmp tmpConfig

	for dcr.More() {
		err = dcr.Decode(&cfgTmp)

		if err != nil {
			return err
		}
	}

	// check if keywords are specified
	if cfgTmp.StTime == "" {
		return fmt.Errorf(`"start time" is not specified`)
	} else if cfgTmp.EdTime == "" {
		return fmt.Errorf(`"end time" is not specified`)
	} else if cfgTmp.GoNum == 0 {
		return fmt.Errorf(`"goroutine num" is not specified`)
	} else if cfgTmp.SourceFile == "" {
		return fmt.Errorf(`"source file" is not specified`)
		// } else if cfgTmp.LogFile == "" {
		// 	return fmt.Errorf(`"Logfile" is not specified`)
	} else if len(cfgTmp.Tasks) == 0 {
		return fmt.Errorf(`"tasks" is not specified`)
	}

	// arc check
	cfg.StTime, err = gnsstime.FromStr(cfgTmp.StTime)

	if err != nil {
		return fmt.Errorf(`invalid time specifed in "start time", %s`, err)
	}

	cfg.EdTime, err = gnsstime.FromStr(cfgTmp.EdTime)

	if err != nil {
		return fmt.Errorf(`invalid time specifed in "end time", %s`, err)
	}

	if cfg.EdTime.LT(cfg.StTime) {
		cfg.StTime, cfg.EdTime = cfg.EdTime, cfg.StTime
	}

	// goroutine num check
	if cfgTmp.GoNum < MinGoroutineNum || cfg.GoNum > MaxGoroutineNum {
		return fmt.Errorf(`invalid goroutine num specifed in "goroutine num", must in %d-%d`,
			MinGoroutineNum, MaxGoroutineNum)
	}

	cfg.GoNum = cfgTmp.GoNum

	// source check
	cfgTmp.SourceFile = filepath.ToSlash(cfgTmp.SourceFile)
	err = parseJsonSource(cfgTmp.SourceFile, NetSourceMap)

	if err != nil {
		return err
	}

	// task check
	numTaskMap := make(map[string]int)
	var val Task
	var ifCheck bool

	for idx, vTmp := range cfgTmp.Tasks {
		if _, ok := NetSourceMap[vTmp.Type]; !ok {
			return fmt.Errorf(`invalid "type" of the %d-th task specified in "tasks"`, idx+1)
		}

		val.Type = vTmp.Type
		val.Path = filepath.ToSlash(vTmp.Path)

		if val.Backward < 0 {
			return fmt.Errorf(`invalid "backward" of the %d-th task specified in "tasks"`, idx+1)
		} else if val.Forward < 0 {
			return fmt.Errorf(`invalid "forward" of the %d-th task specified in "tasks"`, idx+1)
		}

		val.Backward, val.Forward = vTmp.Backward, vTmp.Forward

		if strings.EqualFold(vTmp.IfUnzip, "no") {
			val.IfUnzip = false
		} else {
			val.IfUnzip = true
		}

		ifCheck = false

		if val.IsRnxIGSTask() {
			if vTmp.InfoFile != "" {
				ifCheck = true
				targetInfoMap[val.Type] = new(TargetInfoArray)
				err = targetInfoMap[val.Type].parseJson(vTmp.InfoFile)

				if err != nil {
					return fmt.Errorf(`failed to parse the information file (json) specified in "information" for the %d-th task`, idx+1)
				}
			}

			if len(vTmp.Targets) == 0 {
				for _, site := range targetInfoMap[val.Type].Array {
					TargetListMap[val.Type] = append(TargetListMap[val.Type], strings.ToUpper(site.Name))
				}
			} else {
				for i, site := range vTmp.Targets {
					if ifCheck {
						if j := targetInfoMap[val.Type].Index(site); j != -1 {
							site = targetInfoMap[val.Type].Array[j].Name
						} else {
							return fmt.Errorf(`invalid name "%s" of the %d-th site specified in "sites" for the %d-th task`, site, i+1, idx+1)
						}
					} else if len(site) != 9 {
						return fmt.Errorf(`invalid name "%s" of the %d-th site specified in "sites" for the %d-th task`, site, i+1, idx+1)
					}

					TargetListMap[val.Type] = append(TargetListMap[val.Type], strings.ToUpper(site))
				}
			}
		} else if val.IsCrdILRSTask() {
			for _, sat := range vTmp.Targets {
				TargetListMap[val.Type] = append(TargetListMap[val.Type], sat)
			}
		}

		cfg.Tasks = append(cfg.Tasks, val)

		numTaskMap[val.Type] += 1

		if numTaskMap[val.Type] > 1 {
			return fmt.Errorf(`duplicated "type" of the %d-th task specified in "tasks"`, idx+1)
		}
	}

	// log file check
	cfgTmp.LogFile = filepath.ToSlash(cfgTmp.LogFile)

	if cfgTmp.LogFile != "" {
		if fi, err := os.Stat(filepath.Dir(cfgTmp.LogFile)); os.IsNotExist(err) || !fi.IsDir() {
			return fmt.Errorf(`invalid path specified in "log file"`)
		}

		cfg.LogFile = cfgTmp.LogFile
	}

	// get total number of jobs
	for idx, task := range cfg.Tasks {
		netSource := NetSourceMap[task.Type]
		ts, err := cfg.StTime.SUB(float64(task.Backward))

		if err != nil {
			return fmt.Errorf(`invalid start time for the %d-th task, "backward" may be too large`, idx+1)
		}

		te, err := cfg.EdTime.ADD(float64(task.Forward))

		if err != nil {
			return fmt.Errorf(`invalid end time for the %d-th task, "afterward" may be too large`, idx+1)
		}

		for t, err := ts.NewConvert(netSource.TimeSys); t.LT(te); err = t.AddEq(float64(netSource.Interval)) {
			if err != nil {
				return fmt.Errorf("invalid epoch while processing the %d-th task", idx+1)
			}

			if task.IsRnxIGSTask() || task.IsCrdILRSTask() {
				cfg.JobNum += len(TargetListMap[task.Type])
			} else {
				cfg.JobNum += 1
			}
		}
	}

	return nil
}
