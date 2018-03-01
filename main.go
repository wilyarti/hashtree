package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/BurntSushi/toml"
	"hashtree/downloadFiles"
	"hashtree/hashFiles"
	"hashtree/readDB"
	"hashtree/uploadFiles"
	"hashtree/writeDB"
	"log"
	"os"
	"strings"
	"time"
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
	if len(os.Args) < 3 {
		fmt.Print("Error: please specify bucket and directory!\n")
		os.Exit(1)
	}
	// check for and add trailing / in folder name
	var strs []string
	slash := os.Args[2][len(os.Args[2])-1:]
	var dir = os.Args[2]
	if slash != "/" {
		strs = append(strs, os.Args[2])
		strs = append(strs, "/")
		dir = strings.Join(strs, "")
	}
	// scan files and return map filepath = hash
	var hashmap = make(map[string][]string)
	var remotedb = make(map[string][]string)
	var files = make(map[string][sha256.Size]byte)

	files = hashFiles.Scan(dir)

	// load config to get ready to upload
	var config Config
	config = ReadConfig("/home/undef/.htcfg")
	bucketname := os.Args[1]

	// download .db from server this contains the hashed
	// of all already uploaded files
	// it will be appended to and reuploaded with new hashed at the end
	var dbname []string
	var dbnameLocal []string
	dbname = append(dbname, bucketname)
	dbname = append(dbname, ".db")
	dbnameLocal = append(dbnameLocal, dir)
	dbnameLocal = append(dbnameLocal, strings.Join(dbname, ""))
	downloadlist := make(map[string]string)
	downloadlist[strings.Join(dbname, "")] = strings.Join(dbnameLocal, "")

	// download and check error
	err := downloadFiles.Download(config.Url, config.Port, config.Secure, config.Accesskey, config.Secretkey, config.Enckey, downloadlist, bucketname)
	if err != nil {
		fmt.Println(err)
		fmt.Println("Error .db database is missing, assuming new configuration!")
	} else {
		remotedb, err = readDB.Load(strings.Join(dbnameLocal, ""))
		if err != nil {
			fmt.Println("Error writing database!", err)
			os.Exit(1)
		}
	}

	// build [32]byte => array[ 1, 2, 3 list of files ]
	// of current directory structure
	for file, hash := range files {
		// build local file tree
		s := hex.EncodeToString(hash[:])
		v := hashmap[hex.EncodeToString(hash[:])]
		if len(v) == 0 {
			hashmap[s] = append(hashmap[s], file)
		} else {
			hashmap[s] = append(hashmap[s], file)
		}
	}
	// write database to file
	var hashdb []string
	hashdb = append(hashdb, dir)
	hashdb = append(hashdb, ".")
	hashdb = append(hashdb, os.Args[1])
	hashdb = append(hashdb, ".hsh")

	err = writeDB.Dump(strings.Join(hashdb, ""), hashmap)
	if err != nil {
		fmt.Println("Error writing database!", err)
		os.Exit(1)
	}

	// create map of files for upload
	uploadlist := make(map[string]string)
	for hash, filearray := range hashmap {
		// convert hex to ascii
		// use first file in list for upload
		v := remotedb[hash]
		if filearray[0] == strings.Join(hashdb, "") {
			continue
		} else if filearray[0] == strings.Join(dbnameLocal, "") {
			continue
		} else if len(v) == 0 {
			uploadlist[hash] = filearray[0]
		} else {
			for _, filename := range filearray {
				fmt.Println("[V] ", hash, " => ", filename)
			}
		}

	}
	t := time.Now()
	var reponame []string
	var dbsnapshot []string
	dbsnapshot = append(dbsnapshot, bucketname)
	dbsnapshot = append(dbsnapshot, t.Format("2006-01-02_14:05:05"))
	dbsnapshot = append(dbsnapshot, ".db")

	reponame = append(reponame, bucketname)
	reponame = append(reponame, "-")
	reponame = append(reponame, t.Format("2006-01-02_15:04:05"))
	reponame = append(reponame, ".hsh")

	uploadlist[strings.Join(reponame, "")] = strings.Join(hashdb, "")
	uploadlist[strings.Join(dbname, "")] = strings.Join(dbnameLocal, "")

	// add extra file to remotedb before uploading it
	for file, hash := range files {
		// update remotedb with new files
		s := hex.EncodeToString(hash[:])
		v := remotedb[s]
		if len(v) == 0 {
			remotedb[s] = append(remotedb[s], file)
		} else {
			remotedb[s] = append(remotedb[s], file)
		}

	}
	err = writeDB.Dump(strings.Join(dbnameLocal, ""), remotedb)
	if err != nil {
		fmt.Println("Error writing database!", err)
		os.Exit(1)
	}
	uploadlist[strings.Join(dbsnapshot, "")] = strings.Join(dbnameLocal, "")
	// upload and check error
	err = uploadFiles.Upload(config.Url, config.Port, config.Secure, config.Accesskey, config.Secretkey, config.Enckey, uploadlist, bucketname)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
