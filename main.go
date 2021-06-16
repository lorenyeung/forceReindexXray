package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/lorenyeung/forceReindexXray/auth"
	"github.com/lorenyeung/forceReindexXray/helpers"
	"github.com/lorenyeung/forceReindexXray/internal"

	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh/terminal"
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

	var supportTypesFile helpers.SupportedTypes

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
	var creds auth.Creds
	if flags.ApikeyVar == "" {
		fmt.Println("Enter password or API key: ")
		password, err := terminal.ReadPassword(0)
		if err == nil {
			creds.Apikey = string(password)
		}
	} else {
		creds.Apikey = flags.ApikeyVar
	}

	if flags.UsernameVar == "" || flags.URLVar == "" {
		log.Fatalf("Please specify -user, AND -url flags")
	}

	creds.Username = flags.UsernameVar
	creds.URL = flags.URLVar

	if !auth.VerifyAPIKey(creds.URL, creds.Username, creds.Apikey) {
		log.Fatalf("Please verify your URL and/or credentials. Do not provide context paths in your URL.")
	}
	results := auth.CheckTypeAndRepoParams(creds)

	if flags.ReindexAllVar {
		//index all
		log.Info("Indexing all repos")
		for i := range results {
			log.Info("Indexing ", results[i].Name)
			indexRepo(results[i].Name, results[i].PkgType, supportTypesFile, creds, results[i].Type, flags)
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
					indexRepo(results[j].Name, results[j].PkgType, supportTypesFile, creds, results[j].Type, flags)
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
				indexRepo(flags.RepoVar, results[i].PkgType, supportTypesFile, creds, results[i].Type, flags)
				break
			}
		}
		if !found {
			log.Error("Repo not found in indexed list. ", len(results), " available repos:")
			for i := range results {
				if i < len(results)-1 {
					fmt.Print(results[i].Name, ",")
				} else if i == len(results)-1 {
					fmt.Println(results[i].Name)
				}
			}
		}
	} else {
		log.Error("No repos were specified, please use one of -all, -list or -repo. ", len(results), " available repos:")
		for i := range results {
			if i < len(results)-1 {
				fmt.Print(results[i].Name, ",")
			} else if i == len(results)-1 {
				fmt.Println(results[i].Name)
			}
		}
	}
}

func indexRepo(repo string, pkgType string, types helpers.SupportedTypes, creds auth.Creds, repoType string, flags helpers.Flags) {
	var extensions []helpers.Extensions
	pkgType = strings.ToLower(pkgType)
	log.Debug("type:", repoType, " pkgType:", pkgType, " repo:", repo)
	for i := range types.SupportedPackageTypes {
		if types.SupportedPackageTypes[i].Type == pkgType {
			log.Debug("found package type:", types.SupportedPackageTypes[i].Type)
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
	if repoType == "remote" {
		repo = repo + "-cache"
	}
	fileListData, respCode, _ = auth.GetRestAPI("GET", true, creds.URL+"/artifactory/api/storage/"+repo+flags.FolderVar+"?list&deep=1", creds.Username, creds.Apikey, "", nil, 0)
	if respCode != 200 {
		log.Fatalf("File list received unexpected response code:", respCode, " :", string(fileListData))
	}
	log.Debug("File list received:", string(fileListData))

	var UnindexableMap = make(map[string]int)
	var fileListStruct helpers.FileList
	json.Unmarshal(fileListData, &fileListStruct)
	var notIndexCount, totalCount, notIndexableCount, noExtCount int
	for i := range fileListStruct.Files {
		for j := range extensions {
			fileListStruct.Files[i].Uri = flags.FolderVar + fileListStruct.Files[i].Uri

			log.Debug("File found:", fileListStruct.Files[i].Uri, " matching against:", extensions[j].Extension)
			if strings.Contains(fileListStruct.Files[i].Uri, extensions[j].Extension) {

				if flags.IndexedVar != "" {
					notIndexCount, totalCount = Details(repo, pkgType, types, creds, repoType, flags, fileListStruct.Files[i], notIndexCount, totalCount)
				} else {
					log.Info("File being sent to indexing:", fileListStruct.Files[i].Uri)
					//send to indexing
					m := map[string]string{
						"Content-Type": "application/json",
					}
					body := "{\"artifacts\": [{\"repository\":\"" + repo + "\",\"path\":\"" + fileListStruct.Files[i].Uri + "\"}]}"

					resp, respCode, _ := auth.GetRestAPI("POST", true, creds.URL+"/xray/api/v1/forceReindex", creds.Username, creds.Apikey, body, m, 0)
					if respCode != 200 {
						notIndexCount++
						log.Warn("Unexpected Xray response:HTTP", respCode, " ", string(resp))
					} else {
						log.Info("Xray response:", string(resp))
					}
					totalCount++
				}
				break
			} else if j+1 == len(extensions) {
				//failed the last match
				if flags.LogUnindexableVar {
					log.Info("not indexable:", fileListStruct.Files[i].Uri)
				}
				filePath := strings.Split(fileListStruct.Files[i].Uri, "/")
				fileName := filePath[len(filePath)-1]
				fileExt := strings.Split(fileName, ".")
				notIndexableCount++
				log.Debug("name, name array, uri:", fileName, fileExt, " ", fileListStruct.Files[i].Uri)
				if len(fileExt)-1 > 0 {
					//dont add files without file ext
					UnindexableMap["."+fileExt[len(fileExt)-1]]++
				} else {
					noExtCount++
				}

			}
		}
	}
	log.Info("Total indexed count:", totalCount-notIndexCount, "/", totalCount, " Total not indexable:", notIndexableCount, " Files with no extension:", noExtCount)
	log.Info("Unindexable file types count:", UnindexableMap)
}

func Details(repo string, pkgType string, types helpers.SupportedTypes, creds auth.Creds, repoType string, flags helpers.Flags, fileListData helpers.Files, notIndexCount int, totalCount int) (int, int) {
	//send to details
	var printAll bool
	switch flags.IndexedVar {
	case "unindexed":
	case "all":
		printAll = true
	default:
		log.Fatalf("Please provide one of the following: unindexed all")
	}
	status, proc := internal.GetDetails(repo, pkgType, fileListData.Uri, creds)
	if !proc {
		notIndexCount++
		printStatus(status, repo, pkgType, fileListData.Uri, creds)
	} else {
		totalCount++
		if printAll {
			printStatus(status, repo, pkgType, fileListData.Uri, creds)
		}
	}
	return notIndexCount, totalCount
}

func printStatus(status string, repo string, pkgType string, uri string, creds auth.Creds) {
	var fileDetails []byte
	var fileInfo helpers.FileInfo
	var size string
	if pkgType == "docker" {
		uri = strings.TrimSuffix(uri, "/manifest.json")
		folderDetails, _, _ := auth.GetRestAPI("GET", true, creds.URL+"/artifactory/api/storage/"+repo+uri, creds.Username, creds.Apikey, "", nil, 0)
		json.Unmarshal(folderDetails, &fileInfo)
		var size64 int64
		for i := range fileInfo.Children {
			path := fileInfo.Children[i].Uri
			var fileInfoDocker helpers.FileInfo
			fileDetailsDocker, _, _ := auth.GetRestAPI("GET", true, creds.URL+"/artifactory/api/storage/"+repo+uri+path, creds.Username, creds.Apikey, "", nil, 0)
			json.Unmarshal(fileDetailsDocker, &fileInfoDocker)
			size64 = size64 + helpers.StringToInt64(fileInfoDocker.Size)
		}
		//hardcode for now
		fileInfo.MimeType = "application/json"
		size = helpers.ByteCountDecimal(size64)
	} else {
		fileDetails, _, _ = auth.GetRestAPI("GET", true, creds.URL+"/artifactory/api/storage/"+repo+uri, creds.Username, creds.Apikey, "", nil, 0)
		json.Unmarshal(fileDetails, &fileInfo)
		size = helpers.ByteCountDecimal(helpers.StringToInt64(fileInfo.Size))
	}
	status = fmt.Sprintf("%-19v", status)
	//not really helpful for docker
	log.Info(status, "\t", size, "\t", fmt.Sprintf("%-16v", strings.TrimPrefix(fileInfo.MimeType, "application/")), "\t", repo+uri)
}
