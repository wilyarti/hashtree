package main

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"hashtree/downloadFiles"
	"hashtree/readDB"
	"log"
	"os"
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
	var config Config
	config = ReadConfig("/home/undef/.htcfg")
	bucketname := os.Args[1]
	databasename := os.Args[2]

	// download .db from server this contains the hashed
	var dbnameLocal []string
	dbnameLocal = append(dbnameLocal, dir)
	dbnameLocal = append(dbnameLocal, databasename)
	downloadlist := make(map[string]string)
	downloadlist[databasename] = strings.Join(dbnameLocal, "")

	// download and check error
	var remotedb = make(map[string][]string)
	err := downloadFiles.Download(config.Url, config.Port, config.Secure, config.Accesskey, config.Secretkey, config.Enckey, downloadlist, bucketname)
	if err != nil {
		fmt.Println(err)
		fmt.Println("Error .db database is missing, assuming new configuration!")
		os.Exit(1)
	} else {
		remotedb, err = readDB.Load(strings.Join(dbnameLocal, ""))
		if err != nil {
			fmt.Println("Error writing database!", err)
			os.Exit(1)
		}
	}
	// iterate through hashmap, pull list of file names
	// build these into a hash => path list
	dlist := make(map[string]string)
	for hash, filearray := range remotedb {
		// build local file tree
		for _, file := range filearray {
			dlist[hash] = file
		}
	}
	// Download files
	err = downloadFiles.Download(config.Url, config.Port, config.Secure, config.Accesskey, config.Secretkey, config.Enckey, dlist, bucketname)
	if err != nil {
		os.Exit(1)
	}

}
