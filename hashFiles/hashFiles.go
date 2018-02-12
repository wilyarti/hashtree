package hashFiles

import (
	"crypto/sha256"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

var files = make(map[string][sha256.Size]byte)

func Hash(path string, info os.FileInfo, err error) error {
	if err != nil {
		log.Print(err)
		return nil
	}
	if info.IsDir() {
		return nil
	}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Print(err)
		return nil
	}
	digest := sha256.Sum256(data)
	files[path] = digest

	return nil
}

func Scan(path string) map[string][sha256.Size]byte {
	dir := path
	err := filepath.Walk(dir, Hash)
	if err != nil {
		log.Fatal(err)
	}
	return files
}
