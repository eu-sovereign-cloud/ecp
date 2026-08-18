package crossplane

import "testing"

func TestTranslateImage(t *testing.T) {
	cases := []struct {
		base, version, want string
		wantErr             bool
	}{
		{"ubuntu", "24.04", "ubuntu:24.04", false},
		{"ubuntu", "22.04", "ubuntu:22.04", false},
		{"debian", "12", "debian:12", false},
		{"alma", "9", "almalinux:9", false},
		{"alma", "8", "almalinux:8", false},
		{"windows", "2022", "windows:2022", false},
		{"windows", "2019", "windows:2019", false},
		{"gentoo", "1", "", true},
	}
	for _, c := range cases {
		got, err := translateImage(c.base, c.version)
		if (err != nil) != c.wantErr {
			t.Fatalf("translateImage(%q,%q) err=%v wantErr=%v", c.base, c.version, err, c.wantErr)
		}
		if got != c.want {
			t.Fatalf("translateImage(%q,%q)=%q want %q", c.base, c.version, got, c.want)
		}
	}
}

func TestTranslateZone(t *testing.T) {
	cases := map[string]string{"a": "ZONE_1", "b": "ZONE_2", "c": "AUTO", "": "AUTO", "x": "AUTO"}
	for in, want := range cases {
		if got := translateZone(in); got != want {
			t.Fatalf("translateZone(%q)=%q want %q", in, got, want)
		}
	}
}
