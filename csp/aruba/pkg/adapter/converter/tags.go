package converter

import "slices"

// ArubaTags flattens SECA labels into Aruba tags. A SECA label is a key/value pair while an Aruba
// tag is an opaque free-form string with no structure of its own, so each label is rendered as
// "key=value" - the only encoding that survives the trip without losing the key.
//
// The result is sorted: Go map iteration order is random, and an unsorted slice would make every
// conversion of the same labels produce a different Spec.Tags, which shows up as spurious drift in
// object diffs and makes the converters untestable.
func ArubaTags(labels map[string]string) []string {
	if len(labels) == 0 {
		return nil
	}

	tags := make([]string, 0, len(labels))
	for k, v := range labels {
		tags = append(tags, k+"="+v)
	}
	slices.Sort(tags)

	return tags
}
