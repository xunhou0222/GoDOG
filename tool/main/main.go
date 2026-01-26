/*
  toDOG (TOols for Downloader Of Gnss)
  Copyright (c) 2024-present, Trail of Bits, Inc.
  All rights reserved.

  This source code is licensed in accordance with the terms specified in
  the LICENSE file found in the root directory of this source tree.
*/

package main

import (
	"flag"
	"fmt"
	"todog/igs"
)

func main() {
	// parse config options
	var cfg Config
	var err error

	flag.StringVar(&cfg.SiteFileIGS, "IGS-sites-info", "", "the path of the json file to store the information of IGS sites")
	flag.Parse()

	if cfg.SiteFileIGS != "" {
		err = igs.GetSiteInfoJson(cfg.SiteFileIGS)

		if err != nil {
			panic(fmt.Sprintf("failed to get the json file of IGS sites information, %s", err))
		} else {
			fmt.Println("finished to get the json file of IGS sites information")
		}
	}
}
