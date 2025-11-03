package middleware

import (
	"regexp"
	"testing"
)

func TestGenerateRequestID_UUIDFormat(t *testing.T) {
	// UUID v4 format: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx
	uuidPattern := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

	// Generate multiple UUIDs to ensure consistency
	for i := 0; i < 100; i++ {
		id := generateRequestID()

		// Check length (36 characters with hyphens)
		if len(id) != 36 {
			t.Errorf("UUID length is %d, expected 36: %s", len(id), id)
		}

		// Check format matches UUID v4 pattern
		if !uuidPattern.MatchString(id) {
			t.Errorf("UUID doesn't match v4 format: %s", id)
		}
	}
}

func TestGenerateRequestID_Uniqueness(t *testing.T) {
	seen := make(map[string]bool)
	iterations := 1000

	for i := 0; i < iterations; i++ {
		id := generateRequestID()
		if seen[id] {
			t.Errorf("Duplicate UUID generated: %s", id)
		}
		seen[id] = true
	}

	if len(seen) != iterations {
		t.Errorf("Expected %d unique UUIDs, got %d", iterations, len(seen))
	}
}
func TestGenerateULID_Format(t *testing.T) {
	// ULID should be 26 characters using Crockford's Base32 encoding
	// Valid characters: 0-9 and A-Z (excluding I, L, O, U)
	ulidPattern := regexp.MustCompile(`^[0-9A-HJKMNP-TV-Z]{26}$`)

	for i := 0; i < 10; i++ {
		id := GenerateULID()

		if len(id) != 26 {
			t.Errorf("ULID length is %d, expected 26: %s", len(id), id)
		}

		if !ulidPattern.MatchString(id) {
			t.Errorf("ULID doesn't match Crockford Base32 format: %s", id)
		}
	}
}

func TestGenerateULID_Sortable(t *testing.T) {
	// Generate multiple ULIDs and verify they are lexicographically sorted
	ids := make([]string, 100)
	for i := 0; i < 100; i++ {
		ids[i] = GenerateULID()
		if i > 0 {
			// Each ULID should be >= previous one (lexicographically)
			if ids[i] < ids[i-1] {
				t.Errorf("ULIDs not in order: %s came before %s", ids[i], ids[i-1])
			}
		}
	}
}

func TestGenerateULID_Uniqueness(t *testing.T) {
	seen := make(map[string]bool)
	iterations := 1000

	for i := 0; i < iterations; i++ {
		id := GenerateULID()
		if seen[id] {
			t.Errorf("Duplicate ULID generated: %s", id)
		}
		seen[id] = true
	}

	if len(seen) != iterations {
		t.Errorf("Expected %d unique ULIDs, got %d", iterations, len(seen))
	}
}
