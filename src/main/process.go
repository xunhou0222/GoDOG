package main

import (
	"fmt"
	"godog/crx2rnx"
	"godog/datetime"
	"godog/network"
	"godog/unzip"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

/***** STRUCT **********************************/

type Job struct {
	Type  string
	Time  datetime.Time
	Name  string
	Path  string
	Unzip bool
	Force bool
	Index int
	IsTmp bool
}

/***** VARIABLE ********************************/

var (
	mutexDir sync.Mutex
)

/***** FUNCTION ********************************/

func getPathURL(t datetime.Time, name, template string) string {
	re := regexp.MustCompile(`{([+-]?)(\d?)\.?(\d?)([Rr])}`)
	var flag byte
	var width, precision int
	var hasSpecifiers bool
	var fmtStr, resStr string

	for _, matched := range re.FindAllStringSubmatch(template, -1) {
		hasSpecifiers = false

		if len(matched[1]) != 0 {
			flag = matched[1][0]
			hasSpecifiers = true

			switch flag {
			case '+':
				name = strings.ToUpper(name)
			case '-':
				name = strings.ToLower(name)
			}
		}

		width = 0

		if len(matched[2]) != 0 {
			width, _ = strconv.Atoi(matched[2])
			hasSpecifiers = true
		}

		precision = width

		if len(matched[3]) != 0 {
			precision, _ = strconv.Atoi(matched[2])
			hasSpecifiers = true
		}

		if !hasSpecifiers { // default format
			fmtStr = "%s"
		} else {
			fmtStr = fmt.Sprintf("%%%d.%ds", width, precision)
		}

		resStr = fmt.Sprintf(fmtStr, name)
		template = strings.ReplaceAll(template, matched[0], resStr)
	}

	template = t.Format(template)
	return template
}

/***********************************************/

func doJob(job *Job) (err error) {
	_, err = os.Stat(job.Path)

	if err == nil && !job.Force {
		return io.EOF
	}

	dir := filepath.Dir(job.Path)

	mutexDir.Lock()
	_, err = os.Stat(dir)

	if os.IsNotExist(err) {
		os.MkdirAll(dir, 0775)
	}

	mutexDir.Unlock()

	f := network.NetworkTask{Continue: true}
	var terr network.TaskError
	var srcFile, desFile, extZip string
	var ifZip bool

	job.Index = 0

	for _, s := range rsMap[job.Type].Sources {
		// download
		err = nil
		f.Source.Url = getPathURL(job.Time, job.Name, s.Url)
		f.Source.UserName = s.UserName
		f.Source.Password = s.Password
		f.Path = filepath.ToSlash(filepath.Join(dir, filepath.Base(f.Source.Url)))
		job.Index++

		if f.Source.IsFtp() {
			terr = network.FTPDownload(&f)
		} else if f.Source.IsFtps() {
			terr = network.FTPSDownload(&f)
		} else if f.Source.IsHttpsCddis() {
			terr = network.CDDISDownLoad(&f)
		} else if f.Source.IsHttp() {
			terr = network.HTTPDownload(&f)
		} else if f.Source.IsHttps() {
			terr = network.HTTPDownload(&f)
		} else {
			continue
		}

		if terr != nil {
			job.IsTmp = job.IsTmp || terr.IsTemporary()
			err = terr
			os.Remove(f.Path)
			continue
		}

		// uncompress
		srcFile, desFile = f.Path, f.Path
		err = nil
		extZip = filepath.Ext(srcFile)

		if strings.EqualFold(extZip, ".gz") || strings.EqualFold(extZip, ".Z") {
			ifZip = true
		} else {
			ifZip = false
		}

		if ifZip && job.Unzip {
			if strings.EqualFold(extZip, ".gz") {
				desFile = srcFile[:len(srcFile)-3]
				err = unzip.UnzipGZ(srcFile, desFile)
			} else if strings.EqualFold(extZip, ".Z") {
				desFile = srcFile[:len(srcFile)-2]
				err = unzip.UnzipZ(srcFile, desFile)
			}

			if err != nil {
				os.Remove(srcFile)
				os.Remove(desFile)
				continue
			} else {
				os.Remove(srcFile)
			}
		}

		// convert from crx to rnx
		srcFile = desFile
		err = nil
		ext := job.Time.Format(".{02Y}d")

		if strings.EqualFold(filepath.Ext(srcFile), ".crx") ||
			strings.EqualFold(filepath.Ext(srcFile), ext) {
			desFile = ""
			err = crx2rnx.CRX2RNX(srcFile, &desFile)

			if err != nil {
				os.Remove(srcFile)
				os.Remove(desFile)
				continue
			} else {
				os.Remove(srcFile)
			}
		}

		// rename
		srcFile = desFile
		desFile = job.Path
		err = nil
		ext = filepath.Ext(job.Path)

		if ifZip && !job.Unzip {
			if !strings.EqualFold(ext, extZip) {
				desFile = job.Path + extZip
			}
		}

		err = os.Rename(srcFile, desFile)

		if err != nil {
			os.Remove(srcFile)
			os.Remove(desFile)
			continue
		}

		break
	}

	return
}

/***********************************************/

func process() error {
	GoNumJob := min(jobNum, cfg.GoNum)
	chJobQue := make(chan Job, GoNumJob)
	chJobMsg := make(chan string, GoNumJob)

	// distribute jobs
	go func() {
		var (
			ts, te, dt datetime.Time
			job        Job
			ordInt     int32
			ordDec     float64
		)

		for _, task := range cfg.Tasks {
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
			job.Type, job.Unzip, job.Force, job.Index, job.IsTmp = task.Type, task.IfUnzip, task.IfForce, 0, false

			for job.Time = ts; job.Time.Le(te); job.Time.AddEq(dt) {
				if len(task.Targets) != 0 {
					for _, target := range task.Targets {
						job.Name = target
						job.Path = getPathURL(job.Time, target, task.Path)
						chJobQue <- job
					}
				} else {
					job.Path = getPathURL(job.Time, "", task.Path)
					chJobQue <- job
				}
			}
		}
	}()

	// do jobs
	for i := 0; i < GoNumJob; i++ {
		go func() {
			var count int

			for job := range chJobQue {
				for count = 0; count <= cfg.RetryNum; count++ {
					if err := doJob(&job); err == nil {
						chJobMsg <- fmt.Sprintf("[info] finished to download %s, source index %d, attempt num %d", job.Path, job.Index, count+1)
						break
					} else if err == io.EOF {
						chJobMsg <- fmt.Sprintf("[info] %s already exists", job.Path)
						break
					}
				}

				if count > cfg.RetryNum {
					chJobMsg <- fmt.Sprintf("[ERROR] failed to download %s, attempt num %d", job.Path, count)
				}
			}
		}()
	}

	// write log for each job
	for i := 0; i < jobNum; i++ {
		log.Println(<-chJobMsg)
	}

	return nil
}

/***********************************************/
