package utils

import (
	"errors"
	"math"
)

// GetAverage calculates the average from a slice of int64.
// In Go, slices ([]int64) handle the use cases of both Java Arrays and ArrayLists.
func GetAverage(values []int64) (float64, error) {
	if len(values) == 0 {
		return 0.0, errors.New("empty values")
	}

	var sum int64 = 0
	for _, v := range values {
		sum += v
	}

	// The original Java code only returned the average if sum > 0.
	// We preserve that logic here, though it means negative sums return 0.
	if sum > 0 {
		// Note: The Java code did (sum / values.length), which resulted in integer
		// division (losing the decimal). This is corrected here to float division.
		return float64(sum) / float64(len(values)), nil
	}

	return 0.0, nil
}

// GetAverageFromMap is a placeholder since Go does not support method overloading.
// The original Java code returned null for the Map and long[] overloads.
func GetAverageFromMap(values map[any]any) (float64, error) {
	return 0.0, errors.New("not implemented")
}

// CalculateStandardDeviation calculates the standard deviation.
func CalculateStandardDeviation(values []int64, average float64) float64 {
	if len(values) == 0 {
		return 0.0
	}

	var standardDevOpen float64 = 0.0
	stdOpen := make([]float64, 0, len(values))

	// Note: The original Java code used `for(int i = 1; ...)`, which skipped the
	// first element at index 0. This has been corrected to start at index 0.
	for i := 0; i < len(values); i++ {
		diff := float64(values[i]) - average
		stdOpen = append(stdOpen, math.Pow(diff, 2))
	}

	for _, val := range stdOpen {
		standardDevOpen += val
	}

	standardDevOpen = standardDevOpen / float64(len(stdOpen))
	return math.Sqrt(standardDevOpen)
}

// GetMaxMin calculates the max and min from a slice of int64.
// Go supports multiple return values, making an array return unnecessary.
func GetMaxMin(values []int64) (max int64, min int64) {
	if len(values) == 0 {
		return 0, 0
	}

	min = math.MaxInt64
	// Note: Using math.MinInt64 is safer than initializing to 0
	// in case the array contains only negative numbers.
	max = math.MinInt64

	for _, v := range values {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	return max, min
}
