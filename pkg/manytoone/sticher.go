package manytoone

import (
	"sync"
)

// Stitcher handles the aggregation of data parts and processes them when ready.
type Stitcher[T any] struct {
	stitchData      map[string]T
	requiredParts   []string
	missingParts    map[string]bool              // Track missing parts
	processCallback func(map[string]T, []string) // Callback with missing parts info
	mu              sync.Mutex
}

// NewStitcher creates a new stitcher with required parts and a processing callback.
func NewStitcher[T any](requiredParts []string, processCallback func(map[string]T, []string)) *Stitcher[T] {
	// Initialize missingParts with requiredParts set to true (indicating they are missing initially)
	missingParts := make(map[string]bool)
	for _, part := range requiredParts {
		missingParts[part] = true
	}

	return &Stitcher[T]{
		stitchData:      make(map[string]T),
		requiredParts:   requiredParts,
		missingParts:    missingParts,
		processCallback: processCallback,
	}
}

// AddPart adds a data part to the stitcher and flags missing parts.
func (s *Stitcher[T]) AddPart(partName string, data T) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Store the part in stitchData and mark it as no longer missing
	s.stitchData[partName] = data
	delete(s.missingParts, partName)

	// Collect the remaining missing parts
	var missing []string
	for part := range s.missingParts {
		missing = append(missing, part)
	}

	// Call the processing callback with current data and list of missing parts
	s.processCallback(s.stitchData, missing)
}
