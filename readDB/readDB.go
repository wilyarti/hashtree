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
