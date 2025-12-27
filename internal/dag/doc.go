// Package dag provides a directed acyclic graph implementation for job dependency resolution.
//
// This package is used to:
//   - Build a dependency graph from workflow jobs
//   - Detect circular dependencies
//   - Compute topological ordering for job execution
//
// The implementation uses Kahn's algorithm for topological sorting, which provides
// O(V + E) time complexity where V is the number of jobs and E is the number of
// dependency edges.
package dag
