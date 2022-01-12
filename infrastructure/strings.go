package infrastructure

import "strings"

const loggingUserDataMaxLength = 100

func sanitizeUserLogInput(input string) string {
	var res = input
	res = strings.ReplaceAll(res, "\n", " ")
	res = strings.ReplaceAll(res, "\r", " ")
	if len(res) > loggingUserDataMaxLength {
		res = res[:loggingUserDataMaxLength]
	}
	return res
}
