package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"godog/datetime"
	"log"
	"math"
	"os"
	"path/filepath"
)

/***** CONSTANT ********************************/

const (
	MIN_GOROUTINE_NUM = 1
	MAX_GOROUTINE_NUM = 999
	MIN_RETRY_NUM     = 1
	MAX_RETRY_NUM     = 99
)

/***** STRUCT **********************************/

type Task struct {
	Type     string   `json:"type"`
	Path     string   `json:"path"`
	Backward int      `json:"backward"`
	Forward  int      `json:"forward"`
	IfUnzip  bool     `json:"decompress"`
	IfForce  bool     `json:"force"`
	InfoFile string   `json:"information"`
	Targets  []string `json:"targets"`
}

/***********************************************/

type tConfig struct {
	StTime   string `json:"start time"`
	EdTime   string `json:"end time"`
	GoNum    int    `json:"goroutine num"`
	RetryNum int    `json:"retry num"`
	Tasks    []Task `json:"tasks"`
}

/***********************************************/

type Config struct {
	StTime   datetime.Time
	EdTime   datetime.Time
	GoNum    int
	RetryNum int
	Tasks    []Task
}

/***** FUNCTION ********************************/

func (cfg *Config) ParseJson(cfgFile string) error {
	fp, err := os.Open(cfgFile)

	if err != nil {
		return err
	}

	defer fp.Close()

	dcr := json.NewDecoder(fp)
	var tCfg tConfig

	for dcr.More() {
		err = dcr.Decode(&tCfg)

		if err != nil {
			return err
		}
	}

	// check if keywords are specified
	if tCfg.StTime == "" {
		return errors.New(`no "start time"`)
	} else if tCfg.EdTime == "" {
		return errors.New(`no "end time"`)
	} else if tCfg.GoNum == 0 {
		return errors.New(`no "goroutine num"`)
	} else if tCfg.RetryNum == 0 {
		return errors.New(`no "retry num"`)
	} else if len(tCfg.Tasks) == 0 {
		return errors.New(`no "tasks"`)
	}

	// check the arc
	cfg.StTime = datetime.Str2Time(tCfg.StTime)
	cfg.EdTime = datetime.Str2Time(tCfg.EdTime)

	if cfg.EdTime.Lt(cfg.StTime) {
		return errors.New("invalid arc")
	}

	// check the goroutine num
	if tCfg.GoNum < MIN_GOROUTINE_NUM || tCfg.GoNum > MAX_GOROUTINE_NUM {
		return fmt.Errorf(`value in "goroutine num" must be in %d-%d`, MIN_GOROUTINE_NUM, MAX_GOROUTINE_NUM)
	}

	cfg.GoNum = tCfg.GoNum

	// check the retry num
	if tCfg.RetryNum < MIN_RETRY_NUM || tCfg.RetryNum > MAX_RETRY_NUM {
		return fmt.Errorf(`value in "retry num" must be in %d-%d`, MIN_RETRY_NUM, MAX_RETRY_NUM)
	}

	cfg.RetryNum = tCfg.RetryNum

	// check tasks, and get the total number of jobs
	cfg.Tasks = make([]Task, 0, len(tCfg.Tasks))

	var (
		numTaskMap    = make(map[string]int)
		ts, te, dt, t datetime.Time
		ordInt        int32
		ordDec        float64
	)

	for idx, task := range tCfg.Tasks {
		if _, ok := rsMap[task.Type]; !ok {
			return fmt.Errorf(`invalid "type" of the %d-th task specified in "tasks"`, idx+1)
		}

		task.Path = filepath.ToSlash(task.Path)

		if task.Backward < 0 {
			return fmt.Errorf(`invalid "backward" of the %d-th task specified in "tasks"`, idx+1)
		}

		if task.Forward < 0 {
			return fmt.Errorf(`invalid "forward" of the %d-th task specified in "tasks"`, idx+1)
		}

		if task.InfoFile != "" {
			targetInfoMap[task.Type] = new(TargetInfoArray)
			err = targetInfoMap[task.Type].parseJson(task.InfoFile)

			if err != nil {
				return fmt.Errorf(`failed to parse the information file (json) specified in "information" for the %d-th task`, idx+1)
			}

			if len(task.Targets) == 0 {
				for _, target := range targetInfoMap[task.Type].Array {
					task.Targets = append(task.Targets, target.Name)
				}
			} else {
				targets := make([]string, 0, len(task.Targets))

				for i, target := range task.Targets {
					if j := targetInfoMap[task.Type].Index(target); j >= 0 {
						targets = append(targets, targetInfoMap[task.Type].Array[j].Name)
					} else {
						log.Printf(`[error] the %d-th target "%s" for the %d-th task is not found in its information file, which would be ignored`, i+1, target, idx+1)
						continue
					}
				}

				task.Targets = targets
			}
		}

		numTaskMap[task.Type] += 1

		if numTaskMap[task.Type] > 1 {
			return fmt.Errorf(`duplicated "type" of the %d-th task specified in "tasks"`, idx+1)
		}

		cfg.Tasks = append(cfg.Tasks, task)

		ts = cfg.StTime.Sub(datetime.Seconds2Time(float64(task.Backward)))
		te = cfg.EdTime.Add(datetime.Seconds2Time(float64(task.Forward)))
		ts.ConvertSelf(rsMap[task.Type].TimeSys)
		te.ConvertSelf(rsMap[task.Type].TimeSys)
		dt = datetime.Seconds2Time(float64(rsMap[task.Type].Interval))
		ordDec = float64(int32(ts.OrdTotal()/dt.OrdTotal())) * dt.OrdTotal()
		ordInt = int32(ordDec)
		ordDec -= float64(ordInt)

		if math.Abs(ordDec) < datetime.TIME_EPSILON {
			ordDec = 0
		}

		ts = datetime.Ord2Time(ts.Sys(), ordInt, ordDec)

		for t = ts; t.Le(te); t.AddEq(dt) {
			if len(task.Targets) != 0 {
				jobNum += len(task.Targets)
			} else {
				jobNum += 1
			}
		}
	}

	return nil
}

/***********************************************/
