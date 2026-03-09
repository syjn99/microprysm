package helpers

import (
	"testing"

	"github.com/OffchainLabs/prysm/v7/consensus-types/validator"
	"github.com/OffchainLabs/prysm/v7/testing/assert"
)

// This test verifies how many validator statuses have meaningful values.
// The first expected non-meaningful value will have x.String() equal to its numeric representation.
// This test assumes we start numbering from 0 and do not skip any values.
// Having a test like this allows us to use e.g. `if value < 10` for validity checks.
func TestNumberOfStatuses(t *testing.T) {
	lastValidEnumValue := 12
	x := validator.Status(lastValidEnumValue)
	assert.NotEqual(t, "unknown", x.String())
	x = validator.Status(lastValidEnumValue + 1)
	assert.Equal(t, "unknown", x.String())
}
