package converter

import (
	"slices"
	"strings"
)

// ArubaTags flattens SECA labels into Aruba tags. A SECA label is a key/value pair while an Aruba
// tag is an opaque free-form string with no structure of its own, so each label is rendered as
// "key-value".
//
// The separator is a dash because the Aruba CMP accepts only letters, digits and "-" in a tag, and
// rejects the whole request otherwise:
//
//	400 [semantic] Validation: Tag: character '=' is not valid
//
// That constraint is enforced nowhere else in the stack - the SDK types a tag as a bare string and
// the operator's CRD applies no pattern - so it only shows up against a real backend, as a failed
// resource rather than a rejected field. "=", ":", "_", ".", "/", "+" and space are all refused;
// letters (either case), digits and "-" are accepted.
//
// Everything outside that alphabet is replaced, in the value as well as the key: Kubernetes label
// values legitimately contain "." and "_", and a single unrepresentable character would otherwise
// fail the entire Aruba resource rather than just its tag. A run of rejected characters collapses
// to one dash, and leading and trailing dashes are dropped, so "version=24.04" becomes
// "version-24-04". The mapping is deliberately lossy: it keeps every label visible in the Aruba
// console at the cost of not being reversible, which nothing needs - tags are never read back into
// SECA labels.
//
// The result is sorted and de-duplicated: Go map iteration order is random, an unsorted slice would
// make every conversion of the same labels produce a different Spec.Tags, and sanitising can map
// two distinct labels onto one tag.
func ArubaTags(labels map[string]string) []string {
	if len(labels) == 0 {
		return nil
	}

	tags := make([]string, 0, len(labels))
	for key, value := range labels {
		if tag := sanitizeTag(key + "-" + value); tag != "" {
			tags = append(tags, tag)
		}
	}

	// Labels that sanitise away entirely leave nothing to send, which is the same as having had
	// no labels at all.
	if len(tags) == 0 {
		return nil
	}

	slices.Sort(tags)

	return slices.Compact(tags)
}

// sanitizeTag reduces s to the alphabet the Aruba CMP accepts in a tag: letters, digits and "-".
// Any run of other characters becomes a single dash, and the result carries no leading or trailing
// dash. A string with nothing usable in it yields "", which the caller drops rather than sending an
// empty tag.
func sanitizeTag(s string) string {
	var b strings.Builder
	b.Grow(len(s))

	pendingDash := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			if pendingDash && b.Len() > 0 {
				b.WriteByte('-')
			}
			b.WriteRune(r)
			pendingDash = false
		default:
			// Held back rather than written, so a trailing run leaves no dash behind.
			pendingDash = true
		}
	}

	return b.String()
}
