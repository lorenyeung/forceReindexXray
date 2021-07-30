package helpers

import (
	"flag"
	"fmt"
	"runtime"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

//TraceData trace data struct
type TraceData struct {
	File string
	Line int
	Fn   string
}

type SupportedTypes struct {
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

type FileList struct {
	Files []Files `json:"files"`
}

type Files struct {
	Uri string `json:"uri"`
}

type FileInfo struct {
	Size     string          `json:"size"`
	MimeType string          `json:"mimeType"`
	Children []FileInfoChild `json:"children"`
}

type FileInfoChild struct {
	Uri string `json:"uri"`
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

//ByteCountDecimal convert bytes to human readable data size
func ByteCountDecimal(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(b)/float64(div), "kMGTPE"[exp])
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

//StringToInt64 self explanatory
func StringToInt64(data string) int64 {
	convert, err := strconv.ParseInt(data, 10, 64)
	if err != nil {
		log.Warn(data, " is not of type integer")
		return 0
	}
	return convert
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
	UsernameVar, ApikeyVar, FolderVar, URLVar, RepoVar, LogLevelVar, TypesFileVar, IndexedVar, ListReposVar string
	ReindexAllVar, LogUnindexableVar                                                                        bool
	ReportWorkersVar                                                                                        int
}

//SetFlags function
func SetFlags() Flags {
	var flags Flags
	flag.IntVar(&flags.ReportWorkersVar, "reportWorkers", 5, "Number of indexed report workers")
	flag.StringVar(&flags.IndexedVar, "indexed", "", "Indexed analysis")
	flag.StringVar(&flags.LogLevelVar, "log", "INFO", "Order of Severity: TRACE, DEBUG, INFO, WARN, ERROR, FATAL, PANIC")
	flag.StringVar(&flags.TypesFileVar, "typesFile", "", "supported_types.json file location, get this from Artifactory")
	flag.StringVar(&flags.FolderVar, "folder", "", "Only reindex within a certain folder depth")
	flag.StringVar(&flags.URLVar, "url", "", "Platform URL. No /context")
	flag.StringVar(&flags.UsernameVar, "user", "", "Username")
	flag.StringVar(&flags.ApikeyVar, "apikey", "", "API key or password")

	flag.StringVar(&flags.RepoVar, "repo", "", "Reindex single repo")
	flag.StringVar(&flags.ListReposVar, "list", "", "Reindex list of repos, comma separated. No white space between")
	flag.BoolVar(&flags.ReindexAllVar, "all", false, "Reindex all repos")
	flag.BoolVar(&flags.LogUnindexableVar, "logUnindexable", false, "Log unindexable file types in output")

	flag.Parse()
	return flags
}
