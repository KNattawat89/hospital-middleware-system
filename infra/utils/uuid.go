package utils

import (
	"fmt"

	"github.com/google/uuid"
)

func IsUUID(input string) error {
	_, err := uuid.Parse(input)
	if err != nil {
		return fmt.Errorf("invalid UUID: %w", err)
	}
	return nil
}
