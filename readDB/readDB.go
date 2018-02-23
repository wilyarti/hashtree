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
	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var hashmap = make(map[string][]string)
	var hash string
	for scanner.Scan() {
		matched, err := regexp.MatchString("^--- .*", scanner.Text())
		if err != nil {
			fmt.Println("Error database not comprehensible!!")
			os.Exit(0)
		}
		if matched == true {
			re := regexp.MustCompile("^--- ")
			s := ""
			hash = re.ReplaceAllString(scanner.Text(), s)
			fmt.Println(hash)
			continue
		}
		i := strings.Compare("---", scanner.Text())
		if i == 0 {
			continue
		}
		matched, err = regexp.MatchString("^- .*", scanner.Text())
		if err != nil {
			fmt.Println("Error database not comprehensible!!")
			os.Exit(0)
		}
		if matched == true {
			re := regexp.MustCompile("^- ")
			s := ""
			filepath := re.ReplaceAllString(scanner.Text(), s)
			hashmap[hash] = append(hashmap[hash], filepath)
			fmt.Println(filepath, " added")
			continue
		} else {
			fmt.Println("Error database not comprehensible!!")
			os.Exit(0)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	return hashmap, nil
}
