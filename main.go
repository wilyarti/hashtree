package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hashtree/downloadFiles"
	"hashtree/hashFiles"
	"hashtree/readDB"
	"hashtree/uploadFiles"
	"hashtree/writeDB"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/user"
	//"path"
	"regexp"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
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
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	log.SetFlags(log.Lshortfile)
	// check we have enough command line args
	if len(os.Args) < 3 {
		fmt.Print("Error: please specify bucket and directory!\n")
		os.Exit(1)
	}
	// check for and add trailing / in folder name and add it
	var strs []string
	slash := os.Args[2][len(os.Args[2])-1:]
	var dir = os.Args[2]
	if slash != "/" {
		strs = append(strs, os.Args[2])
		strs = append(strs, "/")
		dir = strings.Join(strs, "")
	}

	// create various variables
	var hashmap = make(map[string][]string)
	var remotedb = make(map[string][]string)
	// create hash database name
	var hashdb []string
	hashdb = append(hashdb, dir)
	hashdb = append(hashdb, ".")
	hashdb = append(hashdb, os.Args[1])
	hashdb = append(hashdb, ".hsh")
	// the default output of files is a byte array and string
	// this is later changed to string[]=>string
	var files = make(map[string][sha256.Size]byte)

	// scan files and return map filepath = hash
	files = hashFiles.Scan(dir)

	// load config to get ready to upload
	// first, find the path of $HOME
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

	// download .db from server this contains the hashed
	// of all already uploaded files
	// it will be appended to and reuploaded with new hashed at the end
	var dbname []string
	var dbnameLocal []string
	dbname = append(dbname, bucketname)
	dbname = append(dbname, ".db")
	dbnameLocal = append(dbnameLocal, dir)
	dbnameLocal = append(dbnameLocal, ".")
	dbnameLocal = append(dbnameLocal, strings.Join(dbname, ""))
	downloadlist := make(map[string]string)
	downloadlist[strings.Join(dbnameLocal, "")] = strings.Join(dbname, "")

	// download and check error
	// download has the format filename => remotename
	err, failedDownload := downloadFiles.Download(config.Url, config.Port, config.Secure, config.Accesskey, config.Secretkey, config.Enckey, downloadlist, bucketname)
	if err != nil {
		for _, file := range failedDownload {
			fmt.Println("Error failed to download: ", file)
		}
		fmt.Println(err)
		fmt.Println("Error .db database is missing, assuming new configuration!")
	} else {
		remotedb, err = readDB.Load(strings.Join(dbnameLocal, ""))
		if err != nil {
			fmt.Println("Error writing database!", err)
		}
	}

	// create out map of [sha256hash] => array of file names
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
	// create map of files for upload
	// do this with the full path of each file before it's
	// modified below.
	var c float64
	uploadlist := make(map[string]string)
	for hash, filearray := range hashmap {
		// convert hex to ascii
		// use first file in list for upload
		v := remotedb[hash]
		// check if database filenames
		if filearray[0] == strings.Join(hashdb, "") {
			continue
		} else if filearray[0] == strings.Join(dbnameLocal, "") {
			continue
			// this file exist remotely
		} else if len(v) == 0 {
			uploadlist[hash] = filearray[0]
			// file exists remotely
		} else {
			c += float64(len(v))
			//for _, _ := range filearray {
			//b := path.Base(filename)
			//fmt.Printf("Parsing database: %v\t %s", c, b)
			//	c++
			//}
		}

	}
	fmt.Println("\nVerified files: ", c)
	// write database to file
	// before writing remove directory prefix
	// so the files in the directory become the root of the data structure
	var hashmapcooked = make(map[string][]string)

	for hash, filearray := range hashmap {
		for _, file := range filearray {
			var reg []string
			reg = append(reg, "^")
			reg = append(reg, dir)
			var re = regexp.MustCompile(strings.Join(reg, ""))
			f := re.ReplaceAllString(file, "")
			hashmapcooked[hash] = append(hashmapcooked[hash], f)

		}
	}
	// add extra file to remotedb before uploading it
	for file, hash := range files {
		// update remotedb with new files
		s := hex.EncodeToString(hash[:])
		v := remotedb[s]
		// remote base name

		if len(v) == 0 {
			remotedb[s] = append(remotedb[s], file)
		} else {
			remotedb[s] = append(remotedb[s], file)
		}

	}

	// upload and check error
	err, failedUploads := uploadFiles.Upload(config.Url, config.Port, config.Secure, config.Accesskey, config.Secretkey, config.Enckey, uploadlist, bucketname)
	if err != nil {
		for _, hash := range failedUploads {
			// remove failed uploads from database
			fmt.Println("Failed to upload: ", hash)
			delete(remotedb, hash)
			delete(hashmapcooked, hash)

		}
		fmt.Println(err)
	}
	// create database and upload
	t := time.Now()
	// create a snapshot of the database
	// create a snapshot of the hash tree
	var reponame []string
	var dbsnapshot []string
	dbsnapshot = append(dbsnapshot, bucketname)
	dbsnapshot = append(dbsnapshot, "")
	dbsnapshot = append(dbsnapshot, t.Format("2006-01-02_14:05:05"))
	dbsnapshot = append(dbsnapshot, ".db")

	reponame = append(reponame, bucketname)
	reponame = append(reponame, "-")
	reponame = append(reponame, t.Format("2006-01-02_15:04:05"))
	reponame = append(reponame, ".hsh")

	// write localdb to hard drive
	err = writeDB.Dump(strings.Join(hashdb, ""), hashmapcooked)
	if err != nil {
		fmt.Println("Error writing database!", err)
		os.Exit(1)
	}

	// write remotedb to file
	err = writeDB.Dump(strings.Join(dbnameLocal, ""), remotedb)
	if err != nil {
		fmt.Println("Error writing database!", err)
		os.Exit(1)
	}

	dbuploadlist := make(map[string]string)
	// add these files to the upload list
	dbuploadlist[strings.Join(reponame, "")] = strings.Join(hashdb, "")
	dbuploadlist[strings.Join(dbname, "")] = strings.Join(dbnameLocal, "")
	dbuploadlist[strings.Join(dbsnapshot, "")] = strings.Join(dbnameLocal, "")
	err, failedUploads = uploadFiles.Upload(config.Url, config.Port, config.Secure, config.Accesskey, config.Secretkey, config.Enckey, dbuploadlist, bucketname)
	if err != nil {
		for _, hash := range failedUploads {
			fmt.Println("Failed to upload: ", hash)
		}
		fmt.Println(err)
	}

	err = os.Remove(strings.Join(hashdb, ""))
	if err != nil {
		fmt.Println("Error deleting database!", err)
	}
	err = os.Remove(strings.Join(dbnameLocal, ""))
	if err != nil {
		fmt.Println("Error deleting database!", err)
	}
}
