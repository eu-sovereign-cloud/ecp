package crossplane

import (
	"fmt"
	"regexp"
	"strings"
)

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

// regionName is a SECA region named after an IONOS location: country, city and an optional
// number, as in de-txl or de-fra-2.
var regionName = regexp.MustCompile(`^[a-z]{2}-[a-z]{3}(-[0-9]+)?$`)

// translateLocation resolves a SECA region name to an IONOS location. A region is named
// after its location with "-" for "/" (de-txl is de/txl, de-fra-2 is de/fra/2), so the name
// is a DNS label and can be a hostname. Only the shape is checked: IONOS decides which
// locations exist, so a new one needs no change here. IP blocks are region-bound, so a
// Workspace and a PublicIp must resolve to the same IONOS location for the reserved address
// to attach to an instance's NIC.
func translateLocation(secaRegion string) (string, error) {
	if !regionName.MatchString(secaRegion) {
		return "", fmt.Errorf("unsupported region %q: name a region after its IONOS location, such as de-txl for de/txl", secaRegion)
	}
	return strings.ReplaceAll(secaRegion, "-", "/"), nil
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
