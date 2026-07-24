package utils

import (
	"regexp"
)

var (
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	slugRegex  = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
)

func ValidateEmail(email string) bool {
	return emailRegex.MatchString(email)
}

func ValidateSlug(slug string) bool {
	return slugRegex.MatchString(slug)
}

func GenerateSlug(title string) string {
	slug := toLowerAndHyphen(title)
	slug = removeNonAlphanumeric(slug)
	slug = trimExtraHyphens(slug)
	return slug
}

func toLowerAndHyphen(s string) string {
	result := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		if c == ' ' || c == '_' {
			result = append(result, '-')
		} else {
			result = append(result, c)
		}
	}
	return string(result)
}

func removeNonAlphanumeric(s string) string {
	result := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			result = append(result, c)
		}
	}
	return string(result)
}

func trimExtraHyphens(s string) string {
	result := make([]byte, 0, len(s))
	prevHyphen := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '-' {
			if !prevHyphen && len(result) > 0 {
				result = append(result, c)
				prevHyphen = true
			}
		} else {
			result = append(result, c)
			prevHyphen = false
		}
	}
	if len(result) > 0 && result[len(result)-1] == '-' {
		result = result[:len(result)-1]
	}
	return string(result)
}
