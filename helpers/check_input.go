package helpers

import "regexp"

func RemoveAlphaNum(text string) string {
	reg := regexp.MustCompile("[^a-zA-Z0-9]+")

	processedString := reg.ReplaceAllString(text, "")

	return processedString
}
