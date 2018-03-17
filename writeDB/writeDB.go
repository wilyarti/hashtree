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
package writeDB

import (
	"bytes"
	"os"
)

type Database struct {
	Checksum [32]byte
	Files    []string
}

//var files = make(map[string][sha256.Size]byte)
//var hashmap = make(map[[32]byte][]string)

func Dump(path string, hashMap map[string][]string) error {
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
		buffer.WriteString(hash)
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
