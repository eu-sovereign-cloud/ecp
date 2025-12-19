package labels

import (
	"crypto/sha3"
	"fmt"
)

// ComputeKeyedLabelKey computes the keyed label key for a given original key.
func ComputeKeyedLabelKey(key string) string {
	return fmt.Sprintf("%s%x", KeyedLabelsPrefix, sha3.Sum224([]byte(key)))
}

func KeyedToOriginal(keyedLabels map[string]string, keys []string) map[string]string {
	original := make(map[string]string, len(keys))
	for _, key := range keys {
		computedKey := ComputeKeyedLabelKey(key)
		if value, exists := keyedLabels[computedKey]; exists {
			original[key] = value
		}
	}
	return original
}

func OriginalToKeyed(originalLabels map[string]string) map[string]string {
	keyedLabels := make(map[string]string, len(originalLabels))
	for k, v := range originalLabels {
		computedKey := ComputeKeyedLabelKey(k)
		keyedLabels[computedKey] = v
	}
	return keyedLabels
}
