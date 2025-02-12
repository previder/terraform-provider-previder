package sorters

import "sort"

func SortMapKeys[T any](mapValues map[string]T) []string {
	// Extract keys and sort them to maintain order
	keys := make([]string, 0, len(mapValues))
	for k := range mapValues {
		keys = append(keys, k)
	}
	sort.Strings(keys) // Sort keys to maintain order
	return keys
}
