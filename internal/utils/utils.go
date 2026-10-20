package utils

import "encoding/json"

func ToJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}
