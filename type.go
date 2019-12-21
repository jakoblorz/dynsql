package dynsql

import "database/sql/driver"

type JSONType string

var (
	JSONString  JSONType = "string"
	JSONNumber  JSONType = "number"
	JSONBoolean JSONType = "boolean"
)

func obtainJSONType(v driver.Value) (JSONType, bool) {
	switch v.(type) {
	case int, int16, int32, int64, float32, float64:
		return JSONNumber, true
	case string:
		return JSONString, true
	case bool:
		return JSONBoolean, true
	}
	return "", false
}
