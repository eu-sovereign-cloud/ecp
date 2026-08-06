package crossplane

import "fmt"

// imageAliases maps a SECA image's (base,version) labels to an IONOS image alias.
// POC scope: the public supported images from the SECA image catalog.
var imageAliases = map[string]map[string]string{
	"ubuntu":  {"24.04": "ubuntu:24.04", "22.04": "ubuntu:22.04"},
	"debian":  {"12": "debian:12"},
	"alma":    {"9": "almalinux:9", "8": "almalinux:8"},
	"windows": {"2022": "windows:2022", "2019": "windows:2019"},
}

// translateImage resolves a SECA image (base,version) to an IONOS image alias.
func translateImage(base, version string) (string, error) {
	if byVersion, ok := imageAliases[base]; ok {
		if alias, ok := byVersion[version]; ok {
			return alias, nil
		}
	}
	return "", fmt.Errorf("unsupported image base=%q version=%q", base, version)
}

// locationAliases maps a SECA region name to the exact IONOS location it's deployed to.
// IP blocks are region-bound, so a Workspace and a PublicIp must resolve to the same IONOS
// location for the reserved address to attach to an instance's NIC.
var locationAliases = map[string]string{
	"regionBerlin":    "de/txl",
	"regionFrankfurt": "de/fra",
}

// translateLocation resolves a SECA region name to an IONOS location.
func translateLocation(secaRegion string) (string, error) {
	if location, ok := locationAliases[secaRegion]; ok {
		return location, nil
	}
	return "", fmt.Errorf("unsupported region %q", secaRegion)
}

// translateZone maps a SECA zone to an IONOS availability zone. ENTERPRISE servers
// accept ZONE_1/ZONE_2; anything else falls back to AUTO.
func translateZone(secaZone string) string {
	switch secaZone {
	case "a":
		return "ZONE_1"
	case "b":
		return "ZONE_2"
	default:
		return "AUTO"
	}
}
