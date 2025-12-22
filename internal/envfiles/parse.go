package envfiles

import (
	"bufio"
	"os"
	"strings"
)

// ParseEnvFile parses a GITHUB_ENV file and returns the environment variables.
// The file format supports:
// - Simple KEY=VALUE format
// - Multiline format with heredoc delimiter: KEY<<DELIMITER\nvalue\nDELIMITER
//
// If the file doesn't exist, an empty map is returned without error.
func ParseEnvFile(path string) (map[string]string, error) {
	result := make(map[string]string)

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return result, nil
		}
		return nil, err
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Check for multiline delimiter format: KEY<<DELIMITER
		if idx := strings.Index(line, "<<"); idx != -1 {
			key := line[:idx]
			delimiter := line[idx+2:]

			// Read lines until we hit the delimiter
			var valueLines []string
			for scanner.Scan() {
				valueLine := scanner.Text()
				if valueLine == delimiter {
					break
				}
				valueLines = append(valueLines, valueLine)
			}
			result[key] = strings.Join(valueLines, "\n")
			continue
		}

		// Simple KEY=VALUE format
		if idx := strings.Index(line, "="); idx != -1 {
			key := line[:idx]
			value := line[idx+1:]
			result[key] = value
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// ParsePathFile parses a GITHUB_PATH file and returns the paths to prepend.
// Each line in the file is a path to add.
// If the file doesn't exist, an empty slice is returned without error.
func ParsePathFile(path string) ([]string, error) {
	var result []string

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return result, nil
		}
		return nil, err
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		result = append(result, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// PrependPath prepends new paths to the current PATH value.
// New paths are added to the beginning, separated by colons.
func PrependPath(currentPath string, newPaths []string) string {
	if len(newPaths) == 0 {
		return currentPath
	}

	combined := strings.Join(newPaths, ":")
	if currentPath == "" {
		return combined
	}

	return combined + ":" + currentPath
}
