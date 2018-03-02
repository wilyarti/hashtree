package main

import (
	"fmt"
	"log"
	"os"
	"regexp"

	"github.com/BurntSushi/toml"
	"github.com/minio/minio-go"
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
	if len(os.Args) < 2 {
		fmt.Println("Error: please specify bucket name!")
		fmt.Println("hashlist <bucketname>")
		os.Exit(1)
	}

	// load config to get ready to upload
	var config Config
	config = ReadConfig("/home/undef/.htcfg")
	bucketname := os.Args[1]
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
