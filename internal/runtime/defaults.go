package runtime

// MergeEnv merges multiple environment variable maps in order.
// Later maps override values from earlier maps.
// This is used to merge environment variables in the order:
// workflow -> job -> step
func MergeEnv(maps ...map[string]string) map[string]string {
	// Pre-calculate total size for efficient allocation
	totalSize := 0
	for _, m := range maps {
		totalSize += len(m)
	}

	result := make(map[string]string, totalSize)

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

// DefaultBaseEnv returns default GitHub Actions environment variables.
// These are the base environment variables that GitHub Actions provides.
func DefaultBaseEnv(workspacePath, sha, ref string) map[string]string {
	return map[string]string{
		"CI":               "true",
		"GITHUB_ACTIONS":   "true",
		"GITHUB_WORKSPACE": workspacePath,
		"GITHUB_SHA":       sha,
		"GITHUB_REF":       ref,
	}
}
