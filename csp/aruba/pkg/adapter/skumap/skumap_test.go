package skumap

import "testing"

func TestComputeFlavor(t *testing.T) {
	tests := []struct {
		name      string
		vcpu, ram int
		want      string
		wantErr   bool
	}{
		{"exact small", 1, 2, "CSO1A2", false},
		{"exact mid", 4, 8, "CSO4A8", false},
		{"exact large", 32, 64, "CSO32A64", false},
		{"same cpu different ram", 2, 8, "CSO2A8", false},
		{"unsupported cpu count", 3, 8, "", true},
		{"unsupported ram for cpu", 4, 12, "", true},
		{"zero", 0, 0, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ComputeFlavor(tt.vcpu, tt.ram)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ComputeFlavor(%d,%d) err = %v, wantErr %v", tt.vcpu, tt.ram, err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("ComputeFlavor(%d,%d) = %q, want %q", tt.vcpu, tt.ram, got, tt.want)
			}
		})
	}
}

func TestImageTemplate(t *testing.T) {
	tests := []struct {
		image   string
		want    string
		wantErr bool
	}{
		{"ubuntu-24-04", "LU24-001", false},
		{"debian-12", "DE12-001", false},
		{"alma-8", "alma8", false},
		{"windows-2022", "WS22-001_W2K22_1_0", false},
		{"ubuntu-99-04", "", true},
		{"", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.image, func(t *testing.T) {
			got, err := ImageTemplate(tt.image)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ImageTemplate(%q) err = %v, wantErr %v", tt.image, err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("ImageTemplate(%q) = %q, want %q", tt.image, got, tt.want)
			}
		})
	}
}

func TestStorageType(t *testing.T) {
	tests := []struct {
		iops int64
		want string
	}{
		{0, "Standard"},
		{5000, "Standard"},
		{performanceIOPSFloor - 1, "Standard"},
		{performanceIOPSFloor, "Performance"},
		{20000, "Performance"},
	}
	for _, tt := range tests {
		if got := StorageType(tt.iops); got != tt.want {
			t.Errorf("StorageType(%d) = %q, want %q", tt.iops, got, tt.want)
		}
	}
}
