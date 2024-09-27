package main

import (
	"fmt"
	"godog/config"
	"godog/crx2rnx"
	"godog/gnsstime"
	"godog/network"
	"godog/network/igs"
	"godog/unzip"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type tJob struct {
	Type    string
	Time    gnsstime.GNSSTime
	Name    string
	Path    string
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

func doJob(job tJob) string {
	var err error

	_, err = os.Stat(job.Path)

	if err == nil {
		return fmt.Sprintf("[INFO] %s already exists", job.Path)
	}

	f := network.NetTask{Continue: true}
	dir := filepath.Dir(job.Path)
	var terr network.TaskError
	var srcFile, desFile string

	mutexDir.Lock()
	_, err = os.Stat(dir)

	if os.IsNotExist(err) {
		os.MkdirAll(dir, 0775)
	}

	mutexDir.Unlock()
	err = nil

	for _, s := range config.NetSourceMap[job.Type].Sources {
		// download
		terr = nil
		f.Source.URL = getPathURL(job.Time, job.Name, s.URL)
		f.Source.UserName = s.UserName
		f.Source.Password = s.Password
		f.Path = filepath.ToSlash(filepath.Join(dir, filepath.Base(f.Source.URL)))

		for i := 0; i < 2; i ++ { // try for two times
			if f.Source.IsFtp() {
				terr = network.FTPDownload(&f)
			} else if f.Source.IsFtps() {
				terr = network.FTPSDownload(&f)
			} else if f.Source.IsHttpsCddis() {
				terr = igs.CDDISDownLoad(&f)
			} else if f.Source.IsHttp() {
				terr = network.HTTPDownload(&f)
			} else {
				return fmt.Sprintf("[ERROR] failed to download %s, unsupported type of URL", job.Path)
			}

			if terr != nil && terr.Temporary() {
				continue
			} else {
				break
			}
		}

		if terr != nil {
			err = terr
			os.Remove(f.Path)
			continue
		}

		// uncompress
		srcFile, desFile = f.Path, f.Path
		err = nil

		if filepath.Ext(srcFile) == ".gz" || filepath.Ext(srcFile) == ".Z" {
			if filepath.Ext(srcFile) == ".gz" {
				desFile = srcFile[:len(srcFile) - 3]
				err = unzip.UnzipGZ(srcFile, desFile)
			} else if filepath.Ext(srcFile) == ".Z" {
				desFile = srcFile[:len(srcFile) - 2]
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

		if filepath.Ext(srcFile) == ".crx" || filepath.Ext(srcFile) == ext {
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
		
		err = os.Rename(srcFile, desFile)

		if err != nil {
			os.Remove(srcFile)
			os.Remove(desFile)
			continue
		}

		break
	}

	if err != nil {
		return fmt.Sprintf("[ERROR] failed to donwload %s, %s", job.Path, err)
	}

	return fmt.Sprintf("[INFO] finished to download %s", job.Path)
}

func procTasks() error {
	var chanTask chan error 
	
	if len(cfg.Tasks) < config.MaxGoroutineNum {
		chanTask = make(chan error, len(cfg.Tasks))
	} else {
		chanTask = make(chan error, config.MaxGoroutineNum)
	}
	
	var chanJob chan tJob = make(chan tJob, cfg.GoNum)
	var chanFinish chan string = make(chan string, cfg.GoNum)

	for idx, task := range cfg.Tasks {
		go func(idx int, task config.Task) {
			ts, err := cfg.StTime.SUB(float64(task.Backward))

			if err != nil {
				chanTask <- fmt.Errorf(`invalid start time for the %d-th task, "backward" may be too large`, idx + 1)
				return
			}

			te, err := cfg.EdTime.ADD(float64(task.Forward))

			if err != nil {
				chanTask <- fmt.Errorf(`invalid end time for the %d-th task, "afterward" may be too large`, idx + 1)
				return
			}

			job := tJob{Type: task.Type}

			for t, err := ts.NewConvert(config.NetSourceMap[task.Type].TimeSys); t.LT(te); 
			    err = t.AddEq( float64(config.NetSourceMap[task.Type].Interval) ) {

				if err != nil {
					chanTask <- fmt.Errorf("invalid epoch while processing the %d-th task", idx + 1)
					return
				}

				job.Time = t

				if task.IsRnxIGSTask() {
					for _, site := range cfg.SitesIGS {
						job.Name = site
						job.Path = getPathURL(t, site, task.Path)

						chanJob <- job
					}
				} else {					
					job.Path = getPathURL(t, "", task.Path)

					chanJob <- job
				}
			}

			chanTask <- nil
		}(idx, task)
	}

	// do works
	for i := 0; i < cfg.GoNum; i++ {
		go func() {
			for {
				job := <-chanJob
				chanFinish <- doJob(job)
			}
		}()
	}

	// wait for all jobs to finish
	for i := uint64(0); i < cfg.JobNum; i++ {
		msg := <-chanFinish

		if logger != nil {
			logger.Println(msg)
		}
	}

	// wait for all tasks to finish
	for i := 0; i < len(cfg.Tasks); i++ {
		err := <-chanTask

		if err != nil {
			return err
		}
	}

	return nil
}
