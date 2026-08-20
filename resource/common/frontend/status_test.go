package frontend

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	"github.com/eu-sovereign-cloud/ecp/framework/kernel"

	"github.com/eu-sovereign-cloud/ecp/resource/common/domain"
)

// IPVersionFromAPI is the request boundary. It used to answer "" for a version it did not know,
// which pushed the rejection down to the CRD's enum check and answered the caller with a
// Kubernetes validation message about a field they never wrote.
func TestIPVersionFromAPI(t *testing.T) {
	testCases := []struct {
		name    string
		in      schema.IPVersion
		want    domain.IPVersion
		wantErr bool
	}{
		{name: "ipv4", in: schema.IPVersionIPv4, want: domain.IPVersionIPv4},
		{name: "ipv6", in: schema.IPVersionIPv6, want: domain.IPVersionIPv6},
		{name: "unset is optional on some specs", in: ""},
		{name: "unknown version is rejected", in: schema.IPVersion("IPv9"), wantErr: true},
		{name: "wrong case is still unknown", in: schema.IPVersion("ipv4"), wantErr: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := IPVersionFromAPI(tc.in)
			if tc.wantErr {
				require.Error(t, err)
				var domErr *kernel.Error
				require.ErrorAs(t, err, &domErr)
				assert.Equal(t, kernel.KindValidation, domErr.Kind, "must map to 422, not 500")
				assert.Equal(t, []kernel.ErrorSource{{Name: "version", Value: string(tc.in)}}, domErr.Sources)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
			// Whatever survives the boundary must survive the trip back out.
			assert.Equal(t, tc.in, IPVersionToAPI(got))
		})
	}
}
