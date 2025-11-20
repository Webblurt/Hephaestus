package utils

import (
	"net/url"
	"strconv"
)

func GetDefaultQueryValue(queryParams url.Values, key, defaultValue string) string {
	value := queryParams.Get(key)

	if value == "" {
		return defaultValue
	}

	return value
}

func GetDefaultIntegerQueryValue(queryParams url.Values, key string, defaultValue int) int {
	valueStr := queryParams.Get(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}
