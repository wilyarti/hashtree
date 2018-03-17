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
package readDB

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
)

func Load(path string) (map[string][]string, error) {
	var hashmap = make(map[string][]string)
	file, err := os.Open(path)
	if err != nil {
		return hashmap, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var hash string
	for scanner.Scan() {
		matched, err := regexp.MatchString("^--- .*", scanner.Text())
		if err != nil {
			fmt.Println("Error database not comprehensible!!")
			return hashmap, err
		}
		if matched == true {
			re := regexp.MustCompile("^--- ")
			s := ""
			hash = re.ReplaceAllString(scanner.Text(), s)
			continue
		}
		i := strings.Compare("---", scanner.Text())
		if i == 0 {
			continue
		}
		matched, err = regexp.MatchString("^- .*", scanner.Text())
		if err != nil {
			fmt.Println("Error database not comprehensible!!")
			return hashmap, err
		}
		if matched == true {
			re := regexp.MustCompile("^- ")
			s := ""
			filepath := re.ReplaceAllString(scanner.Text(), s)
			hashmap[hash] = append(hashmap[hash], filepath)
			continue
		} else {
			fmt.Println("Error database not comprehensible!!")
			return hashmap, err
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
		return hashmap, err
	}

	return hashmap, nil
}
