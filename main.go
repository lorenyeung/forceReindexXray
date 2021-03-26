package main

import (
	"flag"
	"fmt"
	"forceReindexXray/helpers"
	"os"

	log "github.com/sirupsen/logrus"
)

var gitCommit string
var version string

func printVersion() {
	fmt.Println("Current build version:", gitCommit, "Current Version:", version)
}

func main() {

	versionFlag := flag.Bool("v", false, "Print the current version and exit")
	flags := helpers.SetFlags()
	helpers.SetLogger(flags.LogLevelVar)

	switch {
	case *versionFlag:
		printVersion()
		return
	}

	var repoMap = make(map[string]int)

	if flags.TypesFileVar != "" {
		credsFile, err := os.Open(flags.TypesFileVar)
		if err != nil {
			log.Error("Invalid creds file:", err)
			os.Exit(0)
		}
		defer credsFile.Close()
	}

	if flags.ReindexAllVar {
		log.Info("Indexing all repos")
		//index all

	} else if flags.ListReposVar != "" {
		//index specified list
		log.Info("Indexing specified list of repos:", flags.ListReposVar)
	} else if flags.RepoVar != "" {
		log.Info("Indexing single repo:", flags.RepoVar)
		repoMap[flags.RepoVar] = 0
		//default, use passed in repo
	} else {
		log.Fatalf("No repos were specified, please use one of -all, -list or -repo")
	}

}
