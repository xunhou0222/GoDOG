package main

import (
	"fmt"
	"godog/config"
	"godog/crx2rnx"
	"godog/gnsstime"
	"godog/network"
	"godog/unzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	maxTryNum     = 5
	maxRetryNum   = 10
	maxGoNumRetry = 20
)

type tJob struct {
	Type  string
	Time  gnsstime.GNSSTime
	Name  string
	Path  string
	Unzip bool
	Num   int
	Index int
	IsTmp bool
}

var (
	mutexDir sync.Mutex
)

func getPathURL(t gnsstime.GNSSTime, SiteName, template string) string {
	template = t.StrFormat(template, 0)

	if len(SiteName) == 9 {
		template = strings.ReplaceAll(template, "<SITE>", strings.ToLower(SiteName[0:4]))
		template = strings.ReplaceAll(template, "<SITE_LONG>", strings.ToUpper(SiteName))
	}

	return template
}

func doJob(job *tJob) (err error) {
	_, err = os.Stat(job.Path)

	if err == nil {
		return io.EOF
	}

	dir := filepath.Dir(job.Path)

	mutexDir.Lock()
	_, err = os.Stat(dir)

	if os.IsNotExist(err) {
		os.MkdirAll(dir, 0775)
	}

	mutexDir.Unlock()

	f := network.NetTask{Continue: true}
	var terr network.TaskError
	var srcFile, desFile, extZip string
	var ifZip bool

	job.Index = 0
	job.Num++

	for _, s := range config.NetSourceMap[job.Type].Sources {
		// download
		err = nil
		f.Source.URL = getPathURL(job.Time, job.Name, s.URL)
		f.Source.UserName = s.UserName
		f.Source.Password = s.Password
		f.Path = filepath.ToSlash(filepath.Join(dir, filepath.Base(f.Source.URL)))
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
		ext := job.Time.StrFormat(".<YY>d", 0)

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

func procTasks() error {
	var GoNumTask, GoNumJob, GoNumReTry int

	if len(cfg.Tasks) < cfg.GoNum {
		GoNumTask = len(cfg.Tasks)
	} else {
		GoNumTask = cfg.GoNum
	}

	if cfg.JobNum < cfg.GoNum {
		GoNumJob = cfg.JobNum
	} else {
		GoNumJob = cfg.GoNum
	}

	if cfg.JobNum < maxGoNumRetry {
		GoNumReTry = cfg.JobNum
	} else {
		GoNumReTry = maxGoNumRetry
	}

	chTaskIdx := make(chan int, GoNumTask)
	chTaskErr := make(chan error, GoNumTask)
	chJobQue := make(chan tJob, GoNumJob)
	chJobReTry := make(chan tJob, cfg.JobNum)
	chJobMsg := make(chan string, GoNumJob)

	go func() {
		for idx := range cfg.Tasks {
			chTaskIdx <- idx
		}
	}()

	for i := 0; i < GoNumTask; i++ {
		idx := <-chTaskIdx
		task := cfg.Tasks[idx]

		go func(idx int, task config.Task) {
			ts, err := cfg.StTime.SUB(float64(task.Backward))

			if err != nil {
				chTaskErr <- fmt.Errorf(`invalid start time for the %d-th task, "backward" may be too large`, idx+1)
				return
			}

			te, err := cfg.EdTime.ADD(float64(task.Forward))

			if err != nil {
				chTaskErr <- fmt.Errorf(`invalid end time for the %d-th task, "afterward" may be too large`, idx+1)
				return
			}

			job := tJob{Type: task.Type, Unzip: task.IfUnzip, Num: 0, Index: 0, IsTmp: false}

			for t, err := ts.NewConvert(config.NetSourceMap[task.Type].TimeSys); t.LT(te); err = t.AddEq(float64(config.NetSourceMap[task.Type].Interval)) {
				if err != nil {
					chTaskErr <- fmt.Errorf("invalid epoch while processing the %d-th task", idx+1)
					return
				}

				job.Time = t

				if task.IsRnxIGSTask() {
					for _, site := range config.SiteListMap[task.Type] {
						job.Name = site
						job.Path = getPathURL(t, site, task.Path)
						chJobQue <- job
					}
				} else {
					job.Path = getPathURL(t, "", task.Path)
					chJobQue <- job
				}
			}

			chTaskErr <- nil
		}(idx, task)
	}

	// do jobs
	for i := 0; i < GoNumJob; i++ {
		go func() {
			for {
				job := <-chJobQue
				err := doJob(&job)

				if err == nil {
					chJobMsg <- fmt.Sprintf("[INFO] finished to download %s, source index %d, attempt num %d", job.Path, job.Index, job.Num)
				} else if err == io.EOF {
					chJobMsg <- fmt.Sprintf("[INFO] %s already exists", job.Path)
				} else {
					chJobReTry <- job
				}
			}
		}()
	}

	// retry
	for i := 0; i < GoNumReTry; i++ {
		go func() {
			for {
				job := <-chJobReTry

				if job.Num < maxTryNum {
					chJobQue <- job
				} else {
					select {
					case job1 := <-chJobQue:
						chJobQue <- job1
						chJobReTry <- job
					default:
						err := doJob(&job)

						if err == nil {
							chJobMsg <- fmt.Sprintf("[INFO] finished to download %s, source index %d, attempt num %d", job.Path, job.Index, job.Num)
						} else if err == io.EOF {
							chJobMsg <- fmt.Sprintf("[INFO] %s already exists", job.Path)
						} else {
							if job.Num < maxRetryNum {
								chJobReTry <- job
							} else {
								chJobMsg <- fmt.Sprintf("[ERROR] failed to download %s, %s attempt num %d", job.Path, err, job.Num)
							}
						}
					}
				}
			}
		}()
	}

	var chanSignal chan byte = make(chan byte, 1)

	// write log for each job
	go func() {
		for i := 0; i < cfg.JobNum; i++ {
			msg := <-chJobMsg

			if logger != nil {
				logger.Println(msg)
			}
		}

		chanSignal <- 1
	}()

	// wait for all tasks to finish
	for i := 0; i < len(cfg.Tasks); i++ {
		err := <-chTaskErr

		if err != nil {
			return err
		}
	}

	// wait for all jobs to finish
	<-chanSignal

	return nil
}
