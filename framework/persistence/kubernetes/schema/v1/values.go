package v1

// Defines values for IPVersion.
const (
	IPVersionIPv4 IPVersion = "IPv4"
	IPVersionIPv6 IPVersion = "IPv6"
)

// IPVersion IP version of the address
type IPVersion string

// Cidr Combined IPv4 and IPv6 CIDR block for a subnet. Depending on the network
// configuration, either the IPv4 or IPv6 range can be omitted. So the following
// combinations are possible:
//
// * IPv4 only
// * IPv6 only
// * IPv4 and IPv6 (Dual Stack)
type Cidr struct {
	// Ipv4 IPv4 CIDR block for the subnet.
	// +kubebuilder:validation:MaxLength=18
	// +kubebuilder:validation:MinLength=9
	// +kubebuilder:validation:XValidation:rule="self.size() == 0 || (isCIDR(self) && cidr(self).ip().family() == 4)",message="cidr.ipv4 must be a valid IPv4 CIDR block"
	Ipv4 string `json:"ipv4,omitempty" x-cel-message-0:"cidr.ipv4 must be a valid IPv4 CIDR block" x-cel-rule-0:"self.size() == 0 || (isCIDR(self) && cidr(self).ip().family() == 4)" x-kubebuilder-validation-max-length:"18" x-kubebuilder-validation-min-length:"9"`

	// Ipv6 IPv6 CIDR block for the subnet.
	// +kubebuilder:validation:MaxLength=43
	// +kubebuilder:validation:MinLength=4
	// +kubebuilder:validation:XValidation:rule="self.size() == 0 || (isCIDR(self) && cidr(self).ip().family() == 6)",message="cidr.ipv6 must be a valid IPv6 CIDR block"
	Ipv6 string `json:"ipv6,omitempty" x-cel-message-0:"cidr.ipv6 must be a valid IPv6 CIDR block" x-cel-rule-0:"self.size() == 0 || (isCIDR(self) && cidr(self).ip().family() == 6)" x-kubebuilder-validation-max-length:"43" x-kubebuilder-validation-min-length:"4"`
}

// Zone Reference to a specific zone within a region
type Zone = string

// VolumeReference Represents a connection between a Block Storage and a user of the block storage.
type VolumeReference struct {
	// DeviceRef Reference to the block storage used to store the volume.
	DeviceRef Reference `json:"deviceRef"`

	// Type The connection type depends on the type of device and type of block storage.
	// +kubebuilder:default="virtio"
	// +kubebuilder:validation:Enum=virtio
	// +kubebuilder:validation:MaxLength=7
	Type VolumeReferenceType `json:"type,omitempty" x-kubebuilder-default:"virtio" x-kubebuilder-validation-enum:"virtio" x-kubebuilder-validation-max-length:"7"`
}

// VolumeReferenceType The connection type depends on the type of device and type of block storage.
type VolumeReferenceType string
