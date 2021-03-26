# forceReindexXray go script

## Purpose
Re-index Xray Repositories due to bad/incomplete indexing. Tested for 3.x only. Do not recommend running in parallel. 

## Installation
### Standalone Binary
See the new releases section :) 
Those can be run with `./reindex-<DISTRO>-<ARCH>`

### Source code method
Find your go home (`go env`) 
then install under `$GO_HOME/src` (do not create another folder)
`$ git clone https://github.com/lorenyeung/forceReindexXray.git`
then run
`$ go run $GO_HOME/src/forceReindexXray/main.go`

Happy downloading! :)

## Usage
### Commands
* all
    - Description:
        - Re-index all repositories that are currently set for indexing. Must provide this or -list or -repo.
    - Example:
        - ./reindex -all

* apikey (required)
    - Description:
    	- API key or password
    - Example:
        - ./reindex -apikey mypassword

* folder 
    - Description:
        - Optional folder depth in case you don't want to index a whole repository
    - Example:
        - ./reindex -folder /com/google

* list
    - Description:
    	- Provide a list of repositories to re-index. Do not provide -cache for remotes. Comma separated list with no spaces. Must provide this or -all or -repo.
    - Example:
        - ./reindex -list npm-local,jcenter,docker-local

* log
    - Description:
    	- Log level. Order of Severity: TRACE, DEBUG, INFO, WARN, ERROR, FATAL, PANIC (default "INFO")
    - Example: 
        - ./reindex -log debug

* repo
    - Description:
    	- Re-index a single Repository. Must provide this or -list or -all.
    - Example:
        - ./reindex -repo npm-local

* typesfile (required)
    - Description:
    	- supported_types.json file location, get this from Artifactory
    - Example:
        - ./reindex -typesfile support_types.json

* url (required)
    - Description:
    	- Platform URL. Do not provide a context path or trailing slash.
    - Example:
        - ./reindex -url https://loren.devops.io

* user (required)
    - Description:
    	- Username
    - Example:
        - ./reindex -user loren

* v	
    - Description:
        - Print the current version and exit
    - Example:
        - ./reindex -v
