package main

import "strings"

func ternary[T any](condition bool, trueValue, falseValue T) T {
	if condition {
		return trueValue
	}
	return falseValue
}

// convertFileSrc removes file:// prefix and adds /file/ prefix to create a new source URL.
func convertFileSrc(filePath string) string {
	return "/file/" + strings.TrimPrefix(filePath, "file://")
}
