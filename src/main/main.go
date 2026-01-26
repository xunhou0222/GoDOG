/*
GoDOG (Golang Downloader Of Gnss)

Copyright (c) 2024-present JIANG Tingwei.
All rights reserved.

This source code is licensed in accordance with the terms specified in
the LICENSE file found in the root directory of this source tree.
*/
package main

import (
	"flag"
	"log"
)

/***** VARIABLE ********************************/

var (
	rsMap         map[string]Resource         = make(map[string]Resource)
	targetInfoMap map[string]*TargetInfoArray = make(map[string]*TargetInfoArray)
	cfg           Config
	jobNum        int
)

/***** FUNCTION ********************************/

func main() {
	log.Println("[info] GoDOG started")

	// 1. parse command-line options
	var rsFile, cfgFile string
	flag.StringVar(&rsFile, "rs", "./resource.json", "the path of the resource file (json)")
	flag.StringVar(&cfgFile, "cfg", "./config.json", "the path of the config file (json)")
	flag.Parse()

	// 2. parse the resource file
	log.Println("[info] parsing the resource file (json)...")

	if err := ParseResourceJson(rsFile, rsMap); err != nil {
		log.Fatalln("[fatal] error in the resource file (json).", err)
	}

	log.Println("[info] finished parsing the resource file (json)")

	// 3. parse the config file
	log.Println("[info] parsing the config file (json)...")

	if err := cfg.ParseJson(cfgFile); err != nil {
		log.Fatalln("[fatal] error in the config file (json).", err)
	}

	log.Println("[info] finished parsing the config file (json)")
	log.Println("[info] job num:", jobNum)

	// 4. process
	if err := process(); err != nil {
		log.Fatalln("[fatal] error in processing tasks.", err)
	}

	log.Println("[info] finished")
}

/***********************************************/
