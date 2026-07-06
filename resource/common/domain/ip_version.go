package domain

// IPVersion represents the IP address family of a public IP.
type IPVersion string

const (
	IPVersionIPv4 IPVersion = "IPv4"
	IPVersionIPv6 IPVersion = "IPv6"
)
