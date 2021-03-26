package helpers

import (
	"encoding/json"
	"flag"
	"fmt"
	"forceReindexXray/auth"
	"runtime"
	"strings"

	log "github.com/sirupsen/logrus"
)

//TraceData trace data struct
type TraceData struct {
	File string
	Line int
	Fn   string
}

//SetLogger sets logger settings
func SetLogger(logLevelVar string) {
	level, err := log.ParseLevel(logLevelVar)
	if err != nil {
		level = log.InfoLevel
	}
	log.SetLevel(level)

	log.SetReportCaller(true)
	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	customFormatter.QuoteEmptyFields = true
	customFormatter.FullTimestamp = true
	customFormatter.CallerPrettyfier = func(f *runtime.Frame) (string, string) {
		repopath := strings.Split(f.File, "/")
		function := strings.Replace(f.Function, "go-pkgdl/", "", -1)
		return fmt.Sprintf("%s\t", function), fmt.Sprintf(" %s:%d\t", repopath[len(repopath)-1], f.Line)
	}

	log.SetFormatter(customFormatter)
	fmt.Println("Log level set at ", level)
}

//Check logger for errors
func Check(e error, panicCheck bool, logs string, trace TraceData) {
	if e != nil && panicCheck {
		log.Error(logs, " failed with error:", e, " ", trace.Fn, " on line:", trace.Line)
		panic(e)
	}
	if e != nil && !panicCheck {
		log.Warn(logs, " failed with error:", e, " ", trace.Fn, " on line:", trace.Line)
	}
}

//Trace get function data
func Trace() TraceData {
	var trace TraceData
	pc, file, line, ok := runtime.Caller(1)
	if !ok {
		log.Warn("Failed to get function data")
		return trace
	}

	fn := runtime.FuncForPC(pc)
	trace.File = file
	trace.Line = line
	trace.Fn = fn.Name()
	return trace
}

//Flags struct
type Flags struct {
	UsernameVar, ApikeyVar, FolderVar, URLVar, RepoVar, LogLevelVar, TypesFileVar, ListReposVar string
	ReindexAllVar                                                                               bool
}

//SetFlags function
func SetFlags() Flags {
	var flags Flags
	flag.StringVar(&flags.LogLevelVar, "log", "INFO", "Order of Severity: TRACE, DEBUG, INFO, WARN, ERROR, FATAL, PANIC")
	flag.StringVar(&flags.TypesFileVar, "typesfile", "", "supported_types.json file location, get this from Artifactory")
	flag.StringVar(&flags.FolderVar, "folder", "", "Only reindex within a certain folder depth")
	flag.StringVar(&flags.URLVar, "url", "", "Platform URL. No /context")
	flag.StringVar(&flags.UsernameVar, "user", "", "Username")
	flag.StringVar(&flags.ApikeyVar, "apikey", "", "API key or password")

	flag.StringVar(&flags.RepoVar, "repo", "", "Reindex single repo")
	flag.StringVar(&flags.ListReposVar, "list", "", "Reindex list of repos, comma separated. No white space between")
	flag.BoolVar(&flags.ReindexAllVar, "all", false, "Reindex all repos")

	flag.Parse()
	return flags
}

type IndexedRepo struct {
	Name    string `json:"name"`
	PkgType string `json:"pkgType"`
	Type    string `json:"type"`
}

//Test if remote repository exists and is a remote
func CheckTypeAndRepoParams(creds auth.Creds) []IndexedRepo {
	repoCheckData, repoStatusCode, _ := auth.GetRestAPI("GET", true, creds.URL+"/artifactory/api/xrayRepo/getIndex", creds.Username, creds.Apikey, "", nil, 1)
	if repoStatusCode != 200 {
		log.Fatalf("Repo list does not exist.")
	}
	var result []IndexedRepo
	json.Unmarshal(repoCheckData, &result)
	return result
}
