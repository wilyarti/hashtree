package main

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"hashtree/downloadFiles"
	"hashtree/readDB"
	"log"
	"os"
	"os/user"
	"strings"
)

// Info from config file
type Config struct {
	Url       string
	Port      int
	Secure    bool
	Accesskey string
	Secretkey string
	Enckey    string
}

// Reads info from config file
func ReadConfig(configfile string) Config {
	_, err := os.Stat(configfile)
	if err != nil {
		log.Fatal("Config file is missing: ", configfile)
	}
	var config Config
	if _, err := toml.DecodeFile(configfile, &config); err != nil {
		fmt.Println("error")
		log.Fatal(err)
	}
	//log.Print(config.Index)
	return config
}

func main() {
	log.SetFlags(log.Lshortfile)
	if len(os.Args) < 4 {
		fmt.Println("Error: please specify snapshot name and directory!")
		fmt.Println("hashseed <bucketname> <snapshotname> <directory>")
		os.Exit(1)
	}
	// check for and add trailing / in folder name
	var strs []string
	slash := os.Args[3][len(os.Args[3])-1:]
	var dir = os.Args[3]
	if slash != "/" {
		strs = append(strs, os.Args[3])
		strs = append(strs, "/")
		dir = strings.Join(strs, "")
	}

	// load config to get ready to upload
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	var config Config
	var configName []string
	configName = append(configName, usr.HomeDir)
	configName = append(configName, "/.htcfg")
	config = ReadConfig(strings.Join(configName, ""))

	bucketname := os.Args[1]
	databasename := os.Args[2]

	// download .db from server this contains the hashed
	var dbnameLocal []string
	dbnameLocal = append(dbnameLocal, dir)
	dbnameLocal = append(dbnameLocal, databasename)
	downloadlist := make(map[string]string)
	downloadlist[strings.Join(dbnameLocal, "")] = databasename

	// download and check error
	var remotedb = make(map[string][]string)
	err, _ = downloadFiles.Download(config.Url, config.Port, config.Secure, config.Accesskey, config.Secretkey, config.Enckey, downloadlist, bucketname)
	if err != nil {
		fmt.Println("Error unable to download database:", err)
	} else {
		remotedb, err = readDB.Load(strings.Join(dbnameLocal, ""))
		if err != nil {
			fmt.Println("Error processing database!", err)
			os.Exit(1)
		}
	}
	// iterate through hashmap, pull list of file names
	// build these into a hash => path list
	dlist := make(map[string]string)
	for hash, filearray := range remotedb {
		// build local file tree
		for _, file := range filearray {
			var f []string
			f = append(f, dir)
			f = append(f, file)
			dlist[strings.Join(f, "")] = hash
		}
	}
	// Download files
	err, failedDownloads := downloadFiles.Download(config.Url, config.Port, config.Secure, config.Accesskey, config.Secretkey, config.Enckey, dlist, bucketname)
	if err != nil {
		for _, file := range failedDownloads {
			fmt.Println("Error failed to download: ", file)
		}
		os.Exit(1)
	}
}
