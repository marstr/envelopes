package json

import (
	"encoding/json"
	"fmt"

	"github.com/marstr/envelopes"
)

func unmarshalObject(payload []byte, destination interface{}, objectType string, id envelopes.ID, version uint) error {
	if err := json.Unmarshal(payload, destination); err != nil {
		return fmt.Errorf("decode %s object %s (JSONV%d): %w", objectType, id, version, err)
	}
	return nil
}
