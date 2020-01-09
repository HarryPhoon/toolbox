package utils

import (
	"sort"
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
