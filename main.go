/* Copyright <2018> <Wilyarti Howard>
*
* Redistribution and use in source and binary forms, with or without modification, are permitted provided that the following conditions are met:
*
* 1. Redistributions of source code must retain the above copyright notice, this list of conditions and the following disclaimer.
*
* 2. Redistributions in binary form must reproduce the above copyright notice, this list of conditions and the following disclaimer in the documentatio
* n and/or other materials provided with the distribution.
*
* 3. Neither the name of the copyright holder nor the names of its contributors may be used to endorse or promote products derived from this software w
* ithout specific prior written permission.
*
* THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
* IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE
* LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE
* GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRIC
* T LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SU
* CH DAMAGE.
 */
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
	"regexp"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/wilyarti/minio-go"
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
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}
	switch opt := os.Args[1]; opt {
	case "init":
		err := initRepo()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		os.Exit(0)
	case "list":
		hashlist()
		os.Exit(0)
	case "pull":
		hashseed(false)
		os.Exit(0)
	case "nuke":
		hashseed(true)
		os.Exit(0)
	case "push":
		hashtree()
		os.Exit(0)
	default:
		fmt.Println(os.Args[0])
		usage()
		os.Exit(1)
	}

}
func hashlist() {
	log.SetFlags(log.Lshortfile)
	if len(os.Args) < 3 {
		usage()
		os.Exit(1)
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
	bucketname := os.Args[2]
	// New returns an Amazon S3 compatible client object. API compatibility (v2 or v4) is automatically
	// determined based on the Endpoint value.
	s3Client, err := minio.New(config.Url, config.Accesskey, config.Secretkey, config.Secure)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	// Create a done channel to control 'ListObjects' go routine.
	doneCh := make(chan struct{})

	// Indicate to our routine to exit cleanly upon return.
	defer close(doneCh)

	// List all objects from a bucket-name with a matching prefix.
	for object := range s3Client.ListObjects(bucketname, "", config.Secure, doneCh) {
		if object.Err != nil {
			fmt.Println(object.Err)
			return
		}
		matched, err := regexp.MatchString(".hsh$", object.Key)
		if err != nil {
			fmt.Println(err)
		}
		if matched == true {
			fmt.Println(object.Key)
		}
	}
}
func hashseed(nuke bool) {
	log.SetFlags(log.Lshortfile)
	if len(os.Args) < 5 {
		usage()
		os.Exit(1)
	}
	// check for and add trailing / in folder name
	var strs []string
	slash := os.Args[4][len(os.Args[4])-1:]
	var dir = os.Args[4]
	if slash != "/" {
		strs = append(strs, os.Args[4])
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

	bucketname := os.Args[2]
	databasename := os.Args[3]

	// download .db from server this contains the hashed
	var dbnameLocal []string
	dbnameLocal = append(dbnameLocal, dir)
	dbnameLocal = append(dbnameLocal, databasename)
	downloadlist := make(map[string]string)
	downloadlist[strings.Join(dbnameLocal, "")] = databasename

	// download and check error
	var remotedb = make(map[string][]string)
	err, _ = downloadFiles.Download(config.Url, config.Port, config.Secure, config.Accesskey, config.Secretkey, config.Enckey, downloadlist, bucketname, nuke)
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
	err, failedDownloads := downloadFiles.Download(config.Url, config.Port, config.Secure, config.Accesskey, config.Secretkey, config.Enckey, dlist, bucketname, nuke)
	if err != nil {
		for _, file := range failedDownloads {
			fmt.Println("Error failed to download: ", file)
		}
		os.Exit(1)
	}
}
func hashtree() {
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	log.SetFlags(log.Lshortfile)
	// check we have enough command line args
	if len(os.Args) < 4 {
		usage()
		os.Exit(1)
	}
	// check for and add trailing / in folder name and add it
	var strs []string
	slash := os.Args[3][len(os.Args[3])-1:]
	var dir = os.Args[3]
	if slash != "/" {
		strs = append(strs, os.Args[3])
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
	hashdb = append(hashdb, os.Args[2])
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
	bucketname := os.Args[2]

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
	err, failedDownload := downloadFiles.Download(config.Url, config.Port, config.Secure, config.Accesskey, config.Secretkey, config.Enckey, downloadlist, bucketname, false)
	if err != nil {
		for _, file := range failedDownload {
			fmt.Println("Error failed to download: ", file)
		}
		fmt.Println(err)
		fmt.Println("Error: Unable to download database.")
		fmt.Println("\t Hash the database been initialised?")
		os.Exit(1)
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
		remotedb[s] = removeDuplicates(remotedb[s])

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
	dbsnapshot = append(dbsnapshot, "-")
	dbsnapshot = append(dbsnapshot, t.Format("2006-01-02_15:04:05"))
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
func initRepo() error {
	log.SetFlags(log.Lshortfile)
	if len(os.Args) < 3 {
		usage()
		os.Exit(1)
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
	bucketname := os.Args[2]
	// New returns an Amazon S3 compatible client object. API compatibility (v2 or v4) is automatically
	// determined based on the Endpoint value.
	s3Client, err := minio.New(config.Url, config.Accesskey, config.Secretkey, config.Secure)
	if err != nil {
		log.Fatalln(err)
	}

	found, err := s3Client.BucketExists(bucketname)
	if err != nil {
		return err
	}

	if found {
		fmt.Println("Bucket exists.")
	} else {
		fmt.Println("Creating bucket.")
		err = s3Client.MakeBucket(bucketname, "us-east-1")
		if err != nil {
			log.Fatalln(err)
		}
	}
	var strs []string
	slash := os.Args[3][len(os.Args[3])-1:]
	var dir = os.Args[3]
	if slash != "/" {
		strs = append(strs, os.Args[3])
		strs = append(strs, "/")
		dir = strings.Join(strs, "")
	}
	var dbname []string
	var dbnameLocal []string
	dbname = append(dbname, bucketname)
	dbname = append(dbname, ".db")
	dbnameLocal = append(dbnameLocal, dir)
	dbnameLocal = append(dbnameLocal, ".")
	dbnameLocal = append(dbnameLocal, strings.Join(dbname, ""))
	file, err := os.Create(strings.Join(dbnameLocal, ""))
	defer file.Close()
	if err != nil {
		return err
	}
	dbuploadlist := make(map[string]string)
	// add these files to the upload list
	dbuploadlist[strings.Join(dbname, "")] = strings.Join(dbnameLocal, "")
	err, failedUploads := uploadFiles.Upload(config.Url, config.Port, config.Secure, config.Accesskey, config.Secretkey, config.Enckey, dbuploadlist, bucketname)
	if err != nil {
		for _, hash := range failedUploads {
			fmt.Println("Failed to upload: ", hash)
		}
		return err
	}

	err = os.Remove(strings.Join(dbnameLocal, ""))
	if err != nil {
		fmt.Println("Error deleting database!", err)
	}
	return nil

}

func usage() {
	fmt.Println(`Usage:
	Initialise Repository:
		hashtree init <repository> <directory>
	List snapshots:
		hashtree list <repository>
	Deploy snapshot:
		hashtree pull <repository> <snapshot> <directory>
	Overwrite local files:
		hashtree nuke <repository> <snapshot> <directory>
	Create snapshot:
		hashtree push <repository> <directory>`)
}
func removeDuplicates(elements []string) []string {
	// Use map to record duplicates as we find them.
	encountered := map[string]bool{}
	result := []string{}

	for v := range elements {
		if encountered[elements[v]] == true {
			// Do not add duplicate.
		} else {
			// Record this element as an encountered element.
			encountered[elements[v]] = true
			// Append to result slice.
			result = append(result, elements[v])
		}
	}
	// Return the new slice.
	return result
}
