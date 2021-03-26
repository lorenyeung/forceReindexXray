package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"forceReindexXray/auth"
	"forceReindexXray/helpers"
	"io/ioutil"
	"os"
	"strings"

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
	var supportTypesFile supportedTypes

	if flags.TypesFileVar == "" {
		log.Fatalf("Please provide types file")
	}
	if flags.TypesFileVar != "" {
		credsFile, err := os.Open(flags.TypesFileVar)
		if err != nil {
			log.Fatalf("Invalid creds file:", err)
		}
		defer credsFile.Close()
		scanner, _ := ioutil.ReadAll(credsFile)
		json.Unmarshal(scanner, &supportTypesFile)
	}

	if flags.ApikeyVar == "" || flags.UsernameVar == "" || flags.URLVar == "" {
		log.Fatalf("Please specify -user, -apikey AND -url flags")
	}
	var creds auth.Creds
	creds.Apikey = flags.ApikeyVar
	creds.Username = flags.UsernameVar
	creds.URL = flags.URLVar
	if !auth.VerifyAPIKey(creds.URL, creds.Username, creds.Apikey) {
		log.Fatalf("Please verify your URL and/or credentials. Do not provide context paths in your URL.")
	}

	if flags.ReindexAllVar {
		log.Info("Indexing all repos")
		//index all

	} else if flags.ListReposVar != "" {
		//index specified list
		log.Info("Indexing specified list of repos:", flags.ListReposVar)
	} else if flags.RepoVar != "" {
		log.Info("Indexing single repo:", flags.RepoVar)
		results := helpers.CheckTypeAndRepoParams(creds)
		var found bool
		for i := range results {
			if results[i].Name == flags.RepoVar {
				log.Info("Repo is in indexed list")
				found = true
				indexRepo(flags.RepoVar, results[i].PkgType, supportTypesFile, creds, results[i].Type)
				break
			}
		}
		if !found {
			log.Fatalf("Repo not found in indexed list")
		}
		//do something here

		repoMap[flags.RepoVar] = 0
		//default, use passed in repo
	} else {
		log.Fatalf("No repos were specified, please use one of -all, -list or -repo")
	}

}

func indexRepo(repo string, pkgType string, types supportedTypes, creds auth.Creds, repoType string) {
	var extensions []Extensions
	pkgType = strings.ToLower(pkgType)
	log.Debug("type:", repoType, " pkgType:", pkgType, " repo:", repo)
	for i := range types.SupportedPackageTypes {
		if types.SupportedPackageTypes[i].Type == pkgType {
			log.Info("found package type:", types.SupportedPackageTypes[i].Type)
			extensions = types.SupportedPackageTypes[i].Extension
		}
	}

	var repoMap = make(map[string]bool)
	for y := range extensions {
		repoMap[extensions[y].Extension] = true
		log.Debug("Extension added to list:", extensions[y].Extension)
	}
	var fileListData []byte
	if repoType == "local" {
		fileListData, _, _ = auth.GetRestAPI("GET", true, creds.URL+"/artifactory/api/storage/"+repo+"?list&deep=1", creds.Username, creds.Apikey, "", nil, 0)
	} else if repoType == "remote" {
		fileListData, _, _ = auth.GetRestAPI("GET", true, creds.URL+"/artifactory/api/storage/"+repo+"-cache?list&deep=1", creds.Username, creds.Apikey, "", nil, 0)
	}

	log.Debug("File list received:", string(fileListData))

	var fileListStruct fileList
	json.Unmarshal(fileListData, &fileListStruct)

	for i := range fileListStruct.Files {
		for j := range extensions {
			log.Debug("File found:", fileListStruct.Files[i].Uri, " matching against:", extensions[j].Extension)
			if strings.Contains(fileListStruct.Files[i].Uri, extensions[j].Extension) {
				log.Info("File being sent to indexing:", fileListStruct.Files[i].Uri)
				//send to indexing
				break
			}
		}

	}

}

type supportedTypes struct {
	SupportedPackageTypes []SupportedPackageType `json:"supportedPackageTypes"`
}

type SupportedPackageType struct {
	Type      string       `json:"type"`
	Extension []Extensions `json:"extensions"`
}

type Extensions struct {
	Extension string `json:"extension"`
	IsFile    bool   `json:"is_file"`
}

type fileList struct {
	Files []files `json:"files"`
}

type files struct {
	Uri string `json:"uri"`
}
