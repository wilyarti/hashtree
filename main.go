package main

import (
	"crypto/sha256"
	"encoding/hex"

	"fmt"
	"hashtree/hashFiles"
	"hashtree/uploadFiles"
	"hashtree/writeDB"
	//"io/ioutil"
	"github.com/BurntSushi/toml"
	"log"
	"os"
	"strings"
	"time"
	/*//#"gopkg.in/yaml.v2"*/)

var files = make(map[string][sha256.Size]byte)

var hashmap = make(map[[32]byte][]string)

// Info from config file
type Config struct {
	Url       string
	Port      int
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
	// check for trailing /
	var strs []string
	slash := os.Args[2][len(os.Args[2])-1:]
	var dir = os.Args[2]
	if slash != "/" {
		strs = append(strs, os.Args[2])
		strs = append(strs, "/")
		dir = strings.Join(strs, "")
	}
	// check if it is
	// scan files and return map filepath = hash
	files = hashFiles.Scan(dir)
	for file, hash := range files {
		v := hashmap[hash]
		if len(v) == 0 {
			hashmap[hash] = append(hashmap[hash], file)

		} else {
			hashmap[hash] = append(hashmap[hash], file)
		}
	}
	// write database to file
	var hashdb []string
	hashdb = append(hashdb, dir)
	hashdb = append(hashdb, ".")
	hashdb = append(hashdb, os.Args[1])
	hashdb = append(hashdb, ".hsh")

	err := writeDB.Dump(strings.Join(hashdb, ""), hashmap)
	if err != nil {
		fmt.Println("Error writing database!", err)
		os.Exit(1)
	}
	// load config to get ready to upload
	var config Config
	config = ReadConfig("/home/undef/.htcfg")
	bucketname := os.Args[1]
	// upload files
	// create map of files
	uploadlist := make(map[string]string)
	for hash, filearray := range hashmap {
		// convert hex to ascii
		uploadlist[hex.EncodeToString(hash[:])] = filearray[0]
	}
	t := time.Now()
	uploadlist[t.Format("2006-01-02_15:04:05")] = strings.Join(hashdb, "")
	// upload and check error
	err = uploadFiles.Upload(config.Url, config.Port, config.Accesskey, config.Secretkey, config.Enckey, uploadlist, bucketname)
	if err != nil {
		fmt.Println(err)
	}
}
