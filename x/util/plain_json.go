package util

func ToPlainValues(m map[string]interface{}) []interface{} {
	vals := []interface{}{}
	for _, v := range m {
		vals = append(vals, v)
	}
	return vals
}
