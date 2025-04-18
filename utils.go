package main

// Generic function to convert a slice of any type to []string
func map2[T any, T2 any](items []T, convertFn func(T) T2) []T2 {
	result := make([]T2, len(items))
	for i, item := range items {
		result[i] = convertFn(item)
	}
	return result
}
