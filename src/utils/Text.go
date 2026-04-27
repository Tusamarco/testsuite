package utils

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"

	// Note: You must run `go get golang.org/x/text` to use these encodings.
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// ToString extracts the string representation of an object.
// Replicates Java's o.toString() fallback.
func ToString(o any) string {
	if o == nil {
		return ""
	}
	if s, ok := o.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", o)
}

// ToStringWithDefault extracts string, returning def if o is nil.
func ToStringWithDefault(o any, def string) string {
	if o == nil {
		return def
	}
	return ToString(o)
}

// IsEmpty checks if an object's string representation is empty.
func IsEmpty(o any) bool {
	return o == nil || len(ToString(o)) == 0
}

// IsEmptySlice checks that an entire slice has empty strings.
func IsEmptySlice(s []string) bool {
	if s == nil {
		return true
	}
	for _, v := range s {
		if !IsEmpty(v) {
			return false
		}
	}
	return true
}

// IsBlank checks if an object is nil or contains only whitespace.
func IsBlank(o any) bool {
	return o == nil || len(strings.TrimSpace(ToString(o))) == 0
}

// IsBlankSlice checks that an entire slice has blank strings.
func IsBlankSlice(s []any) bool {
	if s == nil {
		return true
	}
	for _, v := range s {
		strVal := ToString(v)
		if !IsBlank(v) && strVal != "" && strVal != " " {
			return false
		}
	}
	return true
}

// DefaultTo checks for empty/nil and returns the default if found.
func DefaultTo(o any, def any) string {
	if o == nil {
		return ToString(def)
	}
	if s, ok := o.(string); ok && s == "" {
		return ToString(def)
	}
	return ToString(o)
}

// DefaultToEmpty defaults to "" if the object is nil or empty string.
func DefaultToEmpty(o any) string {
	return DefaultTo(o, "")
}

// ToInt converts a string to an int. If null or invalid, default is returned.
func ToInt(val any, deflt int) int {
	if val == nil {
		return deflt
	}
	i, err := strconv.Atoi(ToString(val))
	if err != nil {
		return deflt
	}
	return i
}

// ToInteger replicates returning a nullable Integer (pointer in Go).
func ToInteger(val any, deflt *int) *int {
	if val == nil {
		return nil
	}
	i, err := strconv.Atoi(ToString(val))
	if err != nil {
		return deflt
	}
	return &i
}

// ToLng converts a string to a long (int64).
func ToLng(val any, deflt int64) int64 {
	if val == nil {
		return deflt
	}
	i, err := strconv.ParseInt(ToString(val), 10, 64)
	if err != nil {
		return deflt
	}
	return i
}

// ToLong replicates returning a nullable Long (pointer in Go).
func ToLong(val any, deflt *int64) *int64 {
	if val == nil {
		return nil
	}
	i, err := strconv.ParseInt(ToString(val), 10, 64)
	if err != nil {
		return deflt
	}
	return &i
}

// Delimit concatenates a list of values using specified delimiter, prefix, and suffix.
func Delimit(arr []any, delimiter, prefix, suffix string) string {
	var sb strings.Builder
	for i, v := range arr {
		if i > 0 {
			sb.WriteString(delimiter)
		}
		sb.WriteString(prefix)
		sb.WriteString(ToString(v))
		sb.WriteString(suffix)
	}
	return sb.String()
}

// Replace replaces s1 with s2 in s.
// Go standard library natively supports this via strings.ReplaceAll.
func Replace(s, s1, s2 string) string {
	return strings.ReplaceAll(s, s1, s2)
}

// RemoveAccents removes accented characters (Placeholder matching Java behavior).
func RemoveAccents(o any) string {
	if o == nil {
		return ""
	}
	// The original Java code just uppercased the string as the logic was commented out.
	// Preserving that exact behavior here.
	return strings.ToUpper(ToString(o))
}

// ToProperCase capitalizes the first letter and lowercases the rest.
func ToProperCase(o any) string {
	if o == nil {
		return ""
	}
	s := strings.TrimSpace(ToString(o))
	if len(s) == 0 {
		return ""
	}

	// Convert to rune slice to handle multi-byte characters safely
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	for i := 1; i < len(r); i++ {
		r[i] = unicode.ToLower(r[i])
	}
	return string(r)
}

// CastToStringArray converts []any to []string.
func CastToStringArray(arr []any) []string {
	res := make([]string, len(arr))
	for i, v := range arr {
		res[i] = ToString(v)
	}
	return res
}

// GetMultipleResources fetches strings from a dictionary (replaces Java ResourceBundle).
func GetMultipleResources(rsc map[string]string, keys []string) []string {
	var l []string
	for _, key := range keys {
		l = append(l, rsc[key])
	}
	return l
}

// TrimTo limits a string to maxChars safely using runes.
func TrimTo(s string, maxChars int) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	if len(r) <= maxChars {
		return s
	}
	return string(r[:maxChars])
}

// Capitalize capitalizes the first letter of each space-separated word.
func Capitalize(s string) string {
	words := strings.Split(s, " ")
	for i, w := range words {
		if len(w) > 0 {
			r := []rune(w)
			r[0] = unicode.ToUpper(r[0])
			// Intentionally not lowercasing the rest, to match original Java logic
			words[i] = string(r)
		}
	}
	return strings.Join(words, " ")
}

// GetAsStringSeparated joins a slice of items into a string.
func GetAsStringSeparated(l []any, sep string) string {
	if l == nil || sep == "" {
		return ""
	}
	var sb strings.Builder
	for _, v := range l {
		switch val := v.(type) {
		case string, int, int64, float64:
			sb.WriteString(fmt.Sprintf("%v%s", val, sep))
		}
	}
	return sb.String()
}

// --- Encoding Utilities ---

// CastToUTFNio acts as a basic wrapper to ensure strings are UTF-8.
// Since Go strings are inherently UTF-8, this is mostly a pass-through
// unless an external text decoder is explicitly applied to the byte slice.
func CastToUTFNio(str string, sourceEncoding string) string {
	if sourceEncoding == "UTF-8" {
		return str
	}
	// Simplified fallback (real dynamic charset detection requires x/text module)
	return str
}

// CastToUTF converts ISO-8859-1 strings to UTF-8.
func CastToUTF(isoStr string) string {
	// ISO-8859-1 is a direct 1-to-1 byte to rune map
	b := []byte(isoStr)
	var r []rune
	for _, byteVal := range b {
		r = append(r, rune(byteVal))
	}
	return string(r)
}

// GetUTFStringFromDb converts raw encoded strings from a DB into UTF-8.
// Note: This relies on the "golang.org/x/text" package for specific legacy charsets.
func GetUTFStringFromDb(str string, lang string) string {
	if str == "" || lang == "" {
		return ""
	}

	lang = strings.ToLower(lang)
	var decoder transform.Transformer

	switch lang {
	case "ar":
		decoder = charmap.Windows1256.NewDecoder()
	case "zh":
		decoder = simplifiedchinese.HZGB2312.NewDecoder() // Equivalent to EUC-CN
	case "fr", "en", "es":
		decoder = charmap.ISO8859_1.NewDecoder()
	default:
		return str
	}

	reader := transform.NewReader(bytes.NewReader([]byte(str)), decoder)
	decodedBytes, err := io.ReadAll(reader)
	if err != nil {
		// Fallback to original string if decode fails
		return str
	}

	return string(decodedBytes)
}
