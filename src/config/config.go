package config

import (
	"encoding/json"
	"fmt"
	"godog/gnsstime"
	"godog/network/igs"
	"os"
	"path/filepath"
)

const (
	MinGoroutineNum = 1
	MaxGoroutineNum = 100
)

var (
	SiteArrayIGS igs.SiteInfoArray
	NetSourceMap map[string]NetSource = make(map[string]NetSource)
)

type Config struct {
	StTime    gnsstime.GNSSTime
	EdTime    gnsstime.GNSSTime

	GoNum     int

	LogFile   string

	Tasks     []Task
	SitesIGS  []string

	JobNum    uint64
}

type tmpConfig struct {
	StTime      string    `json:"start time"`
	EdTime      string    `json:"end time"`

	GoNum       int       `json:"goroutine num"`

	SourceFile  string    `json:"source file"`
	IGSFile     string    `json:"IGS file"`
	LogFile     string    `json:"log file"`

	Tasks       []Task    `json:"tasks"`
	SitesIGS    []string  `json:"IGS sites"`
}

func ParseJsonConfig(path string, cfg *Config) error {
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
		return fmt.Errorf(`"start time" is not specified in the config file`)
	} else if cfgTmp.EdTime == "" {
		return fmt.Errorf(`"end time" is not specified in the config file`)
	// } else if cfgTmp.IGSFile == "" {
		// return fmt.Errorf(`"IGS file" is not specified in the config file`)
	} else if cfgTmp.SourceFile == "" {
		return fmt.Errorf(`"source file" is not specified in the config file`)
	// } else if cfgTmp.LogFile == "" {
	// 	return fmt.Errorf(`"Logfile" is not specified in the config file`)
	} else if len(cfgTmp.Tasks) == 0 {
		return fmt.Errorf(`"tasks" is not specified in the config file`)
	// } else if len(cfgTmp.Sites) == 0 {
	// 	return fmt.Errorf(`"tasks" is not specified in the config file`)
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
	err = ParseJsonSource(cfgTmp.SourceFile, NetSourceMap)

	if err != nil {
		return err
	}

	// task check
	ifIGSFile := false
	numTaskMap := make(map[string]int)

	for idx, val := range cfgTmp.Tasks {
		if _, ok := NetSourceMap[val.Type]; !ok {
			return fmt.Errorf(`invalid "type" of the %d-th task specified in "tasks"`, idx + 1)
		}

		if val.Backward < 0 {
			return fmt.Errorf(`invalid "backward" of the %d-th task specified in "tasks"`, idx + 1)
		} else if val.Forward < 0 {
			return fmt.Errorf(`invalid "forward" of the %d-th task specified in "tasks"`, idx + 1)
		}

		val.Path = filepath.ToSlash(val.Path)
		cfg.Tasks = append(cfg.Tasks, val)

		if val.IsRnxIGSTask() {
			ifIGSFile = true
		}

		numTaskMap[val.Type] += 1

		if numTaskMap[val.Type] > 1 {
			return fmt.Errorf(`duplicated "type" of the %d-th task specified in "tasks"`, idx + 1)
		}
	}

	// IGS file check
	cfgTmp.IGSFile = filepath.ToSlash(cfgTmp.IGSFile)

	if ifIGSFile {
		if cfgTmp.IGSFile == "" {
			return fmt.Errorf(`"IGS file" is not specified in the config file`)
		}

		terr := igs.GetJsonIGS(cfgTmp.IGSFile)

		if terr != nil {
			return terr
		}

		err = igs.ParseJsonIGS(cfgTmp.IGSFile, &SiteArrayIGS)

		if err != nil {
			return err
		}
	}

	// sites check
	if ifIGSFile {
		if len(cfgTmp.SitesIGS) == 0 {
			for _, site := range SiteArrayIGS.Array {
				cfg.SitesIGS = append(cfg.SitesIGS, site.Name)
			}
		} else {
			for _, site := range cfgTmp.SitesIGS {
				idx := SiteArrayIGS.Index(site)

				if idx == -1 {
					return fmt.Errorf(`"%s" specified in "sites" is not a valid site`, site)
				}

				cfg.SitesIGS = append(cfg.SitesIGS, SiteArrayIGS.Array[idx].Name)
			}
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

			if task.IsRnxIGSTask() {
				cfg.JobNum += uint64( len(cfg.SitesIGS) )
			} else {
				cfg.JobNum += 1
			}
		}
	}

	return nil
}
