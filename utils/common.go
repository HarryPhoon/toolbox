package utils

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

const (
	idTruncLength = 12
)

func ShortID(id string) string {
	if len(id) > idTruncLength {
		return id[:idTruncLength]
	}
	return id
}

// ReferenceCanBeID checks if the provided text matches a format for an ID
func ReferenceCanBeID(text string) bool {
	matched, err := regexp.MatchString(`^[a-f0-9]\{6,64\}$`, text)
	if err != nil {
		logrus.Error(err)
	}
	return matched
}

// ReferenceHasDomain checks if the provided text has a domain definition in it
func ReferenceHasDomain(text string) bool {
	i := strings.IndexRune(text, '/')
	if i == -1 {
		return false
	}

	// A domain should contain a top level domain name. An exception is 'localhost'
	if strings.ContainsAny(text[:i], ".:") && text[:i] != "localhost" {
		return false
	}

	return true
}

// NumberPrompt creates an interactive prompt that expects and returns an integer
func NumberPrompt(defaultValue int, min int, max int, prompt string) int {
	var tmp string
	var response int = 0

	for true {
		fmt.Printf("%s (%d to abort) [%d-%d]: ", prompt, defaultValue, min, max)
		fmt.Scanf("%s", &tmp)

		tmpresponse, err := strconv.ParseInt(tmp, 10, 0)
		if err != nil {
			continue
		}

		response = int(tmpresponse)

		if response >= min && response <= max {
			break
		}
	}

	return response
}

func JoinJSON(joinkey string, maps ...[]map[string]interface{}) []map[string]interface{} {
	var json []map[string]interface{}
	found := make(map[string]bool)

	// Iterate over every json provided and check if it is already in the final json
	// If it contains some invalid entry (equals nil), then skip that entry

	for _, m := range maps {
		for _, entry := range m {
			if entry["names"] == nil && entry["Names"] == nil {
				continue
			}
			key := entry[joinkey].(string)
			if _, ok := found[key]; !ok {
				found[key] = true
				json = append(json, entry)
			}
		}
	}
	return json
}

func SortJSON(json []map[string]interface{}, key string, hasInterface bool) []map[string]interface{} {
	sort.Slice(json, func(i, j int) bool {
		if hasInterface {
			return json[i][key].([]interface{})[0].(string) < json[j][key].([]interface{})[0].(string)
		}
		return json[i][key].(string) < json[j][key].(string)
	})

	return json
}
