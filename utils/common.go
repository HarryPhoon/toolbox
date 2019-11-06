package utils

import "sort"

const (
	idTruncLength = 12
)

func ShortID(id string) string {
	if len(id) > idTruncLength {
		return id[:idTruncLength]
	}
	return id
}

func JoinJson(joinkey string, maps ...[]map[string]interface{}) []map[string]interface{} {
	var json []map[string]interface{}
	found := make(map[string]bool)

	for _, m := range maps {
		for _, image := range m {
			key := image[joinkey].(string)
			if _, ok := found[key]; !ok {
				found[key] = true
				json = append(json, image)
			}
		}
	}
	return json
}

func SortJson(json []map[string]interface{}, key string, hasInterface bool) []map[string]interface{} {
	sort.Slice(json, func(i, j int) bool {
		if hasInterface {
			return json[i][key].([]interface{})[0].(string) < json[j][key].([]interface{})[0].(string)
		}
		return json[i][key].(string) < json[j][key].(string)
	})

	return json
}
