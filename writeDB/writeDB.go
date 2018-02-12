package writeDB

import (
	"os"
	//"fmt"
	"bytes"
	 "encoding/hex"
)

type Database struct {
	Checksum [32]byte
	Files    []string
}

//var files = make(map[string][sha256.Size]byte)
//var hashmap = make(map[[32]byte][]string)

func Dump(path string, hashMap map[[32]byte][]string) error {
    //Open File or die
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	// create a buffer to make a string as we go along
	var buffer bytes.Buffer

	//// formatter
    for hash, filelist := range hashMap {
        // new entries begin with ---' '
        buffer.WriteString("--- ")
        buffer.WriteString(hex.EncodeToString(hash[:]))
        buffer.WriteString("\n")
        buffer.WriteString("---\n")
        // iterate through array and add file to yaml formatter
        for _, filename := range filelist {
            buffer.WriteString("- ")
            buffer.WriteString(filename)
            buffer.WriteString("\n")
        }
    }

    file.Write(buffer.Bytes())
    //fmt.Println(buffer.String())
	return nil
}
