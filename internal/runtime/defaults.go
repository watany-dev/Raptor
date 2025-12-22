package runtime

// MergeEnv merges multiple environment variable maps in order.
// Later maps override values from earlier maps.
// This is used to merge environment variables in the order:
// workflow -> job -> step
func MergeEnv(maps ...map[string]string) map[string]string {
	result := make(map[string]string)

	for _, m := range maps {
		if m == nil {
			continue
		}
		for key, value := range m {
			result[key] = value
		}
	}

	return result
}
