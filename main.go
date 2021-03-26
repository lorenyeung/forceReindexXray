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
	if flags.FolderVar != "" && !strings.HasPrefix(flags.FolderVar, "/") {
		log.Info("Missing prefix forward slash on folder path, adding in.")
		flags.FolderVar = "/" + flags.FolderVar
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
	results := helpers.CheckTypeAndRepoParams(creds)

	if flags.ReindexAllVar {
		//index all
		log.Info("Indexing all repos")
		for i := range results {
			log.Info("Indexing ", results[i].Name)
			indexRepo(results[i].Name, results[i].PkgType, supportTypesFile, creds, results[i].Type, flags.FolderVar)
		}

	} else if flags.ListReposVar != "" {
		//index specified list
		log.Info("Indexing specified list of repos:", flags.ListReposVar)
		list := strings.Split(flags.ListReposVar, ",")
		for i := range list {
			log.Debug("Removing -cache as needed")
			list[i] = strings.TrimSuffix(list[i], "-cache")
			var found bool
			for j := range results {
				if results[j].Name == list[i] {
					log.Info("Repo is in indexed list:", list[i])
					indexRepo(results[j].Name, results[j].PkgType, supportTypesFile, creds, results[j].Type, flags.FolderVar)
					found = true
					break
				}
			}
			if !found {
				log.Warn(list[i], " was not found in the indexed list, skipping")
			}
		}
	} else if flags.RepoVar != "" {
		//default, use passed in repo
		log.Debug("Removing -cache as needed")
		flags.RepoVar = strings.TrimSuffix(flags.RepoVar, "-cache")
		log.Info("Indexing single repo:", flags.RepoVar)

		var found bool
		for i := range results {
			if results[i].Name == flags.RepoVar {
				log.Info("Repo is in indexed list")
				found = true
				indexRepo(flags.RepoVar, results[i].PkgType, supportTypesFile, creds, results[i].Type, flags.FolderVar)
				break
			}
		}
		if !found {
			log.Fatalf("Repo not found in indexed list")
		}

	} else {
		log.Fatalf("No repos were specified, please use one of -all, -list or -repo")
	}

}

func indexRepo(repo string, pkgType string, types supportedTypes, creds auth.Creds, repoType string, FolderVar string) {
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
	var respCode int
	if repoType == "local" {
		fileListData, respCode, _ = auth.GetRestAPI("GET", true, creds.URL+"/artifactory/api/storage/"+repo+FolderVar+"?list&deep=1", creds.Username, creds.Apikey, "", nil, 0)
	} else if repoType == "remote" {
		fileListData, respCode, _ = auth.GetRestAPI("GET", true, creds.URL+"/artifactory/api/storage/"+repo+"-cache"+FolderVar+"?list&deep=1", creds.Username, creds.Apikey, "", nil, 0)
	}
	if respCode != 200 {
		log.Warn("File list received unexpected response code:", respCode, " :", string(fileListData))
	}

	log.Debug("File list received:", string(fileListData))

	var fileListStruct fileList
	json.Unmarshal(fileListData, &fileListStruct)

	for i := range fileListStruct.Files {
		for j := range extensions {
			fileListStruct.Files[i].Uri = FolderVar + fileListStruct.Files[i].Uri
			log.Debug("File found:", fileListStruct.Files[i].Uri, " matching against:", extensions[j].Extension)
			if strings.Contains(fileListStruct.Files[i].Uri, extensions[j].Extension) {
				log.Info("File being sent to indexing:", fileListStruct.Files[i].Uri)
				//send to indexing
				m := map[string]string{
					"Content-Type": "application/json",
				}
				body := "{\"artifacts\": [{\"repository\":\"" + repo + "\",\"path\":\"" + fileListStruct.Files[i].Uri + "\"}]}"

				resp, respCode, _ := auth.GetRestAPI("POST", true, creds.URL+"/xray/api/v1/forceReindex", creds.Username, creds.Apikey, body, m, 0)
				if respCode != 200 {
					log.Warn("Unexpected Xray response:HTTP", respCode, " ", string(resp))
				} else {
					log.Info("Xray response:", string(resp))
				}
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
