package main

import (
	"regexp"
	"strings"
	"unicode"
)

func PackageName(serviceID string) string {
	pkg := serviceID
	pkg = strings.ToLower(pkg)
	pkg = strings.TrimSpace(pkg)
	pkg = strings.ReplaceAll(pkg, " ", "")
	return pkg
}

func FileName(resourceName string) string {
	file := resourceName
	file = strings.ToLower(file)
	file = strings.TrimSpace(file)
	file = strings.ReplaceAll(file, " ", "_")
	return file
}

func ResourceName(name string) string {
	name = strings.ReplaceAll(name, " ", "")
	return strings.Title(name)
}

func FieldName(name string) string {
	name = replaceInitials(name)
	name = strings.Title(name)
	return name
}

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

func CamelToLowerSnake(name string) string {
	snake := matchFirstCap.ReplaceAllString(name, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

func InputName(name string) string {
	return CamelToLowerSnake(name)
}

func ResourceType(service, resource string) string {
	return "aws:" + strings.ToLower(service) + "_" + CamelToLowerSnake(resource)
}

var initialisms = []string{
	// AWS specific
	"ARN",
	"AWS",
	"CORS",
	"IAM",
	"JWT",
	"SSID",
	"VPC",
	"EBS",

	// From golint
	"ACL", "API", "ASCII", "CPU", "CSS", "DNS", "EOF", "GUID", "HTML", "HTTP",
	"HTTPS", "ID", "IP", "JSON", "LHS", "QPS", "RAM", "RHS", "RPC", "SLA",
	"SMTP", "SQL", "SSH", "TCP", "TLS", "TTL", "UDP", "UI", "UID", "UUID",
	"URI", "URL", "UTF8", "VM", "XML", "XMPP", "XSRF", "XSS",
}

var initMap map[string]bool

func init() {
	initMap = make(map[string]bool)
	for _, init := range initialisms {
		initMap[init] = true
	}
}

// replaceInitials is extracted from golint
// https://github.com/golang/lint/blob/738671d3881b9731cc63024d5d88cf28db875626/lint.go#L716-L764
func replaceInitials(str string) string {
	// Split camelCase at any lower->upper transition, and split on underscores.
	// Check each word for common initialisms.
	runes := []rune(str)
	w, i := 0, 0 // index of start of word, scan
	for i+1 <= len(runes) {
		eow := false // whether we hit the end of a word
		if i+1 == len(runes) {
			eow = true
		} else if runes[i+1] == '_' {
			// underscore; shift the remainder forward over any run of underscores
			eow = true
			n := 1
			for i+n+1 < len(runes) && runes[i+n+1] == '_' {
				n++
			}

			// Leave at most one underscore if the underscore is between two digits
			if i+n+1 < len(runes) && unicode.IsDigit(runes[i]) && unicode.IsDigit(runes[i+n+1]) {
				n--
			}

			copy(runes[i+1:], runes[i+n+1:])
			runes = runes[:len(runes)-n]
		} else if unicode.IsLower(runes[i]) && !unicode.IsLower(runes[i+1]) {
			// lower->non-lower
			eow = true
		}
		i++
		if !eow {
			continue
		}

		// [w,i) is a word.
		word := string(runes[w:i])
		if u := strings.ToUpper(word); initMap[u] {
			// Keep consistent case, which is lowercase only at the start.
			if w == 0 && unicode.IsLower(runes[w]) {
				u = strings.ToLower(u)
			}
			// All the common initialisms are ASCII,
			// so we can replace the bytes exactly.
			copy(runes[w:], []rune(u))
		} else if w > 0 && strings.ToLower(word) == word {
			// already all lowercase, and not the first word, so uppercase the first character.
			runes[w] = unicode.ToUpper(runes[w])
		}
		w = i
	}
	return string(runes)
}
