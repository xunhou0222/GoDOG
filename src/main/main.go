/*
  GoDOG (Golang Downloader Of Gnss)
  Copyright (c) 2024-present, Trail of Bits, Inc.
  All rights reserved.

  This source code is licensed in accordance with the terms specified in
  the LICENSE file found in the root directory of this source tree.
*/

package main

import (
	"flag"
	"fmt"
	"godog/config"
	"log"
	"os"
	"path/filepath"
)

var (
	cfg    config.Config
	logger *log.Logger = nil
)

func main() {
	// get the path of the config file (json), and parse it
	var cfgFile string

	flag.StringVar(&cfgFile, "cfg", "./config.json", "the path of the config file (json)")
	flag.Parse()

	cfgFile = filepath.ToSlash(cfgFile)
	err := cfg.ParseJson(cfgFile)

	if err != nil {
		panic(fmt.Sprintf("[FATAL] failed to parse the config file (json), %s", err))
	}

	// create the file logger
	if cfg.LogFile != "" {
		fpLog, err := os.OpenFile(cfg.LogFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0664)

		if err != nil {
			panic(fmt.Sprintf("[FATAL] failed to create the log file, %s", err))
		}

		defer fpLog.Close()

		logger = log.New(fpLog, "", log.Ldate|log.Ltime)
	}

	fmt.Println("------------------------------------ GoDOG ------------------------------------")

	if logger != nil {
		logger.Writer().Write([]byte("------------------------------------ GoDOG ------------------------------------\n"))
		logger.Printf(`[INFO] finished to parse the config file (json) "%s"`, cfgFile)
		logger.Printf("[INFO] job num: %d", cfg.JobNum)
	}

	// do tasks
	err = procTasks()

	if err != nil {
		msg := fmt.Sprintf("[FATAL] error occured while processing tasks, %s", err)

		if logger != nil {
			logger.Println(msg)
		}

		panic(msg)
	}

	// finished
	fmt.Println("[INFO] finished")

	if logger != nil {
		logger.Println("[INFO] finished")
	}
}
