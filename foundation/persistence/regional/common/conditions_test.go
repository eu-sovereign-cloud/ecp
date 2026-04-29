package common

import (
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/eu-sovereign-cloud/ecp/foundation/persistence/generated/types"
)

func TestEqualConditions(t *testing.T) {
	base := types.StatusCondition{
		Type:             "Ready",
		Reason:           "ReconcileSucceeded",
		Message:          "all good",
		State:            types.ResourceStateActive,
		LastTransitionAt: metav1.Now(),
	}

	cases := []struct {
		name string
		a, b types.StatusCondition
		want bool
	}{
		{
			name: "identical",
			a:    base,
			b:    base,
			want: true,
		},
		{
			name: "different LastTransitionAt is ignored",
			a:    base,
			b: func() types.StatusCondition {
				c := base
				c.LastTransitionAt = metav1.NewTime(base.LastTransitionAt.Add(time.Hour))
				return c
			}(),
			want: true,
		},
		{
			name: "different Type",
			a:    base,
			b: func() types.StatusCondition {
				c := base
				c.Type = "Degraded"
				return c
			}(),
			want: false,
		},
		{
			name: "different Reason",
			a:    base,
			b: func() types.StatusCondition {
				c := base
				c.Reason = "SomethingFailed"
				return c
			}(),
			want: false,
		},
		{
			name: "different Message",
			a:    base,
			b: func() types.StatusCondition {
				c := base
				c.Message = "not so good"
				return c
			}(),
			want: false,
		},
		{
			name: "different State",
			a:    base,
			b: func() types.StatusCondition {
				c := base
				c.State = types.ResourceStateError
				return c
			}(),
			want: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := EqualConditions(tc.a, tc.b); got != tc.want {
				t.Errorf("EqualConditions() = %v, want %v", got, tc.want)
			}
		})
	}
}
