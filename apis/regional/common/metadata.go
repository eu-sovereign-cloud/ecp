package common

// RegionalCommonData defines the additional common fields that can be set on regional resources
type RegionalCommonData struct {
	// Annotations User-defined key/value pairs that are mutable and can be used to add annotations.
	// The number of annotations is eventually limited by the CSP.
	Annotations map[string]string `json:"annotations,omitempty"`

	// Extensions User-defined key/value pairs that are mutable and can be used to add extensions.
	// Extensions are subject to validation by the CSP, and any value that is not accepted will be rejected during admission.
	Extensions map[string]string `json:"extensions,omitempty"`

	// Labels User-defined key/value pairs that are mutable and can be used to
	// organize and categorize resources. They can be used to filter resources.
	// The number of labels is eventually limited by the CSP.
	Labels map[string]string `json:"labels,omitempty"`
}
