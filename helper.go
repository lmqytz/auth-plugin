package main

import (
    "errors"
    "strconv"
)

func ConventToInt64(from interface{}) (int64, error) {
    switch from.(type) {
    case int:
        return int64(from.(int)), nil
    case int32:
        return int64(from.(int32)), nil
    case float32:
        return int64(from.(float32)), nil
    case float64:
        return int64(from.(float64)), nil
    case string:
        tmp, _ := strconv.Atoi(from.(string))
        return int64(tmp), nil
    case []byte:
        tmp, _ := strconv.Atoi(string(from.([]byte)))
        return int64(tmp), nil
    default:
        return 0, errors.New("unknown type")
    }
}
