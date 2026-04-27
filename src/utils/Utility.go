package utils

import (
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var rnd *rand.Rand

const (
	Separator = string(os.PathSeparator)
)

// init initializes the random number generator on package load.
func init() {
	rnd = rand.New(rand.NewSource(time.Now().UnixNano()))
}

// GetListAllFiles recursively retrieves all file paths within a directory.
func GetListAllFiles(rootPath string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(rootPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// CheckFilePath checks if a path exists, and if not, creates it.
func CheckFilePath(path string) bool {
	if path == "" {
		return false
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := os.MkdirAll(path, 0755)
		return err == nil
	}
	return true
}

// CheckEntryInArray checks if a specific value exists in a slice.
func CheckEntryInArray(ar []any, val any) bool {
	for _, o := range ar {
		if o == val {
			return true
		}
	}
	return false
}

// ConvertArgsToMap parses a string slice into a map.
// Supports the custom "@" delimited nested map logic from the original Java.
func ConvertArgsToMap(args []string) map[string]any {
	if len(args) == 0 {
		return nil
	}

	argMap := make(map[string]any)
	hasAtSymbol := false

	for _, arg := range args {
		if strings.Contains(arg, "@") {
			hasAtSymbol = true
			break
		}
	}

	if hasAtSymbol {
		for _, arg := range args {
			parts := strings.SplitN(arg, "@", 2)
			if len(parts) < 2 {
				continue
			}
			sectionName := parts[0]
			kv := strings.SplitN(parts[1], "=", 2)
			if len(kv) < 2 {
				continue
			}
			key, value := kv[0], kv[1]

			// Initialize nested map if not exists
			if _, exists := argMap[sectionName]; !exists {
				argMap[sectionName] = make(map[string]string)
			}

			// Assert and set
			nestedMap := argMap[sectionName].(map[string]string)
			nestedMap[key] = value
		}
	} else {
		for _, arg := range args {
			kv := strings.SplitN(arg, "=", 2)
			if len(kv) == 2 {
				argMap[kv[0]] = kv[1]
			}
		}
	}
	return argMap
}

// ReturnFormatExtension returns file extensions based on an integer.
func ReturnFormatExtension(i int) string {
	switch i {
	case 1:
		return "bmp"
	case 2:
		return "jpg"
	case 3:
		return "tif"
	default:
		return ""
	}
}

// --- Time and Date Helpers ---

func GetYear() string        { return fmt.Sprintf("%04d", time.Now().Year()) }
func GetMonth() string       { return time.Now().Month().String() }
func GetMonthNumber() string { return fmt.Sprintf("%02d", int(time.Now().Month())) }
func GetDayNumber() string   { return fmt.Sprintf("%02d", time.Now().Day()) }
func GetHour() string        { return fmt.Sprintf("%02d", time.Now().Hour()) }
func GetMinute() string      { return fmt.Sprintf("%02d", time.Now().Minute()) }
func GetSecond() string      { return fmt.Sprintf("%02d", time.Now().Second()) }

// GetTimestamp generates a timestamp like: 2026_04_27_14_53_10
func GetTimestamp() string {
	return time.Now().Format("2006_01_02_15_04_05")
}

// GetTimeStampFormatted formats a unix timestamp using a given format string.
// Falls back to a standard format if none provided.
func GetTimeStampFormatted(systemTimeMilli int64, format ...string) string {
	if systemTimeMilli == 0 {
		return ""
	}
	t := time.UnixMilli(systemTimeMilli)
	layout := "2006-01-02 15:04:05" // Default Go layout equivalent to yyyy-MM-dd HH:mm:ss
	if len(format) > 0 {
		// Note: A true Java SimpleDateFormat to Go layout conversion helper
		// is recommended if complex dynamic Java strings are passed.
		layout = format[0]
	}
	return t.Format(layout)
}

// --- File Utilities ---

func DeleteFile(fileName string, objectPath ...string) bool {
	fullPath := fileName
	if len(objectPath) > 0 {
		fullPath = filepath.Join(objectPath[0], fileName)
	}
	err := os.Remove(fullPath)
	return err == nil || os.IsNotExist(err)
}

func DeleteFiles(filesName []string, objectPath string) bool {
	success := true
	for _, f := range filesName {
		if !DeleteFile(f, objectPath) {
			success = false
		}
	}
	return success
}

func CopyFile(source, destinationPath, fileDestName string) bool {
	if !CheckFilePath(filepath.Dir(source)) {
		return false
	}
	if err := os.MkdirAll(destinationPath, 0755); err != nil {
		return false
	}

	destFile := filepath.Join(destinationPath, fileDestName)
	in, err := os.Open(source)
	if err != nil {
		return false
	}
	defer in.Close()

	out, err := os.Create(destFile)
	if err != nil {
		return false
	}
	defer out.Close()

	// io.Copy is highly optimized in Go, avoiding the need for manual byte buffers
	_, err = io.Copy(out, in)
	return err == nil
}

func CreateBackupCopy(fileName string, deleteOriginal bool) bool {
	fileInfo, err := os.Stat(fileName)
	if os.IsNotExist(err) {
		return false
	}

	backupName := fmt.Sprintf("bck_%s_%s_%s_%s", GetDayNumber(), GetMonthNumber(), GetYear(), fileInfo.Name())
	success := CopyFile(fileName, filepath.Dir(fileName), backupName)

	if success && deleteOriginal {
		return DeleteFile(fileName)
	}
	return success
}

// GetArrayListAsDelimitedString joins a slice of strings using a delimiter.
func GetArrayListAsDelimitedString(list []string, delimiter string) string {
	if len(list) == 0 {
		return ""
	}
	return strings.Join(list, delimiter)
}

// GetPhoneNumber generates a random phone number string.
func GetPhoneNumber() string {
	first := GetNumberFromRandomMinMax(1, 700)
	second := GetNumberFromRandomMinMax(100, 999)
	third := GetNumberFromRandomMinMax(1001, 9999)
	return fmt.Sprintf("%d-%d-%d", first, second, third)
}

// --- Random Number Generators ---

func GetNumberFromRandom(index int64) int64 {
	if index <= 0 {
		return 0
	}
	return rnd.Int63n(index)
}

func GetNumberFromRandomMinMax(min, max int64) int64 {
	if min >= max {
		return max
	}
	val := rnd.Int63n(max-min) + min
	if val < 0 {
		val *= -1
	}
	if val < min {
		return min
	}
	return val
}

func GetUnsignNumberFromRandomMinMax(min, max int64) int64 {
	if max > min {
		return rnd.Int63n(max-min) + min
	} else if min > max {
		return rnd.Int63n(min-max) + max
	}
	return max
}

func GetNumberFromRandomMinMaxCeiling(min, max, ceiling int64) int64 {
	if min == max {
		return max
	}
	if min == 0 && max == 0 {
		return 0
	}

	maxL := rnd.Int63()
	if maxL < min {
		maxL = min * 2
	}

	if (maxL - min) > ceiling {
		return int64(math.Abs(float64(min + ceiling)))
	}
	return int64(math.Abs(float64(maxL)))
}

// GetNumberFromUniformRandomMinMax replaces Apache UniformRealDistribution.
func GetNumberFromUniformRandomMinMax(min, max int64) int64 {
	if min >= max {
		return max
	}
	number := rnd.Float64()
	val := int64(math.Round(float64(max) * number))

	if val > max || val < min {
		return GetNumberFromUniformRandomMinMax(min, max)
	}
	return val
}

// GetNumberFromGaussianRandomMinMax replaces Apache NormalDistribution.
func GetNumberFromGaussianRandomMinMax(min, max int64, rng int) int64 {
	if min >= max {
		return max
	}
	number := rnd.NormFloat64() // equivalent to standard normal distribution
	val := int64(math.Round(number*float64(rng) + float64(max)/2.0))

	if val > max || val < min {
		return GetNumberFromGaussianRandomMinMax(min, max, rng)
	}
	return val
}

// GetNumberFromParetoRandomMinMax replaces Apache ParetoDistribution.
func GetNumberFromParetoRandomMinMax(min, max int64, seedIn float64) int64 {
	if min >= max {
		return max
	}

	shape := seedIn
	if shape <= 0 {
		shape = 1.0 // Prevent divide by zero
	}

	// Inverse Transform Sampling for Pareto Distribution
	u := rnd.Float64()
	number := 1.0 / math.Pow(1.0-u, 1.0/shape)

	if math.Round(number) > 0 {
		number = number - math.Round(number)
	}
	number = math.Abs(number)

	val := int64(math.Round(float64(max) * number))
	if val > max || val < min {
		return GetNumberFromParetoRandomMinMax(min, max, seedIn)
	}
	return val
}

// --- Validation and Formatting Utilities ---

func IsNumeric(str string) bool {
	_, err := strconv.ParseInt(str, 10, 64)
	if err == nil {
		return true
	}
	// The original Java code only evaluated Long parsing for `isNumeric`.
	return false
}

func IsDouble(str string) bool {
	_, err := strconv.ParseFloat(str, 64)
	return err == nil
}

func IsPositiveLong(n int64) bool {
	return n >= 0
}

func IsPositiveFloat(n float64) bool {
	return n >= 0
}

func IsEvenNumber(n int64) bool {
	return n%2 == 0
}

// FormatStringToPrint right-aligns strings by prepending spaces.
func FormatStringToPrint(length int, s string) string {
	// %*s is Go's built-in formatting for dynamic width padding
	return fmt.Sprintf("%*s", length, s)
}

// FormatNumberToPrint right-aligns numbers by prepending spaces.
func FormatNumberToPrint(length int, number any) string {
	return fmt.Sprintf("%*s", length, fmt.Sprint(number))
}
