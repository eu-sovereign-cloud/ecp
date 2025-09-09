package filter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatchLabels(t *testing.T) {
	testCases := []struct {
		name             string
		labels           map[string]string
		selector         string
		expectMatch      bool
		expectK8sHandled bool
		expectErr        bool
	}{
		{
			name:             "empty selector matches anything",
			labels:           map[string]string{"env": "prod"},
			selector:         "",
			expectMatch:      true,
			expectK8sHandled: false,
			expectErr:        false,
		},
		{
			name:             "simple equality match",
			labels:           map[string]string{"env": "prod"},
			selector:         "env=prod",
			expectMatch:      false,
			expectK8sHandled: true,
			expectErr:        false,
		},
		{
			name:             "simple equality with double equals operator",
			labels:           map[string]string{"env": "prod"},
			selector:         "env==prod",
			expectMatch:      false,
			expectK8sHandled: true,
			expectErr:        false,
		},
		{
			name:             "simple equality mismatch",
			labels:           map[string]string{"env": "dev"},
			selector:         "env=prod",
			expectMatch:      false,
			expectK8sHandled: true,
			expectErr:        false,
		},
		{
			name:             "multiple selectors match",
			labels:           map[string]string{"env": "prod", "tier": "backend"},
			selector:         "env=prod,tier=backend",
			expectMatch:      false,
			expectK8sHandled: true,
			expectErr:        false,
		},
		{
			name:             "key wildcard match",
			labels:           map[string]string{"region-east-1": "true"},
			selector:         "region-*=true",
			expectMatch:      true,
			expectK8sHandled: false,
			expectErr:        false,
		},
		{
			name:             "value wildcard match",
			labels:           map[string]string{"version": "v1.2.3-alpha"},
			selector:         "version=v1.2.*",
			expectMatch:      true,
			expectK8sHandled: false,
			expectErr:        false,
		},
		{
			name:             "key and value wildcard match",
			labels:           map[string]string{"app.kubernetes.io/version": "0.1.0"},
			selector:         "app.kubernetes.io/version=*.*.0",
			expectMatch:      true,
			expectK8sHandled: false,
			expectErr:        false,
		},
		{
			name:             "numeric greater than match",
			labels:           map[string]string{"replicas": "5"},
			selector:         "replicas>3",
			expectMatch:      true,
			expectK8sHandled: false,
			expectErr:        false,
		},
		{
			name:             "numeric greater than mismatch",
			labels:           map[string]string{"replicas": "3"},
			selector:         "replicas>3",
			expectMatch:      false,
			expectK8sHandled: false,
			expectErr:        false,
		},
		{
			name:             "numeric less than or equal match",
			labels:           map[string]string{"cpu": "0.5"},
			selector:         "cpu<=0.5",
			expectMatch:      true,
			expectK8sHandled: false,
			expectErr:        false,
		},
		{
			name:             "numeric comparison on non-numeric label value",
			labels:           map[string]string{"cpu": "high"},
			selector:         "cpu>1",
			expectMatch:      false,
			expectK8sHandled: false,
			expectErr:        false,
		},
		{
			name:             "namespaced key match",
			labels:           map[string]string{"monitoring.cloud.google.com/scrape": "true"},
			selector:         "monitoring.cloud.google.com/scrape=true",
			expectMatch:      false,
			expectK8sHandled: true,
			expectErr:        false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			match, k8sHandled, err := MatchLabels(tc.labels, tc.selector)

			if tc.expectErr {
				assert.Error(t, err)
				assert.False(t, match)
				assert.False(t, k8sHandled)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectMatch, match, "match should match expectMatch")
				assert.Equal(t, tc.expectK8sHandled, k8sHandled, "k8sHandled should match expectK8sHandled")
			}
		})
	}
}

func TestK8sSelectorForAPI(t *testing.T) {
	testCases := []struct {
		name          string
		rawSelector   string
		expectK8sSafe string
	}{
		{
			name:          "empty selector",
			rawSelector:   "",
			expectK8sSafe: "",
		},
		{
			name:          "only simple equality",
			rawSelector:   "env=prod,tier=backend",
			expectK8sSafe: "env=prod,tier=backend",
		},
		{
			name:          "mixed simple and complex selectors",
			rawSelector:   "env=prod,replicas>5,tier!=frontend,*version*=v1",
			expectK8sSafe: "env=prod,tier!=frontend",
		},
		{
			name:          "one complex, one simple selector",
			rawSelector:   "replicas>5,tier!=frontend",
			expectK8sSafe: "tier!=frontend",
		},
		{
			name:          "selector with spaces",
			rawSelector:   " env = prod , tier!=backend",
			expectK8sSafe: "env = prod,tier!=backend",
		},
		{
			name:          "selector with double equals",
			rawSelector:   "env==prod",
			expectK8sSafe: "env==prod",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			safeSelector := K8sSelectorForAPI(tc.rawSelector)
			assert.Equal(t, tc.expectK8sSafe, safeSelector)
		})
	}
}

func TestWildcardMatch(t *testing.T) {
	testCases := []struct {
		name     string
		pattern  string
		s        string
		expected bool
	}{
		{"exact match", "abc", "abc", true},
		{"exact mismatch", "abc", "def", false},
		{"star only", "*", "anything", true},
		{"prefix match", "a*", "abc", true},
		{"prefix mismatch", "b*", "abc", false},
		{"suffix match", "*c", "abc", true},
		{"suffix mismatch", "*d", "abc", false},
		{"substring match", "*b*", "abc", true},
		{"substring mismatch", "*d*", "abc", false},
		{"multiple wildcards", "a*c*e", "abracadabra-e", true},
		{"multiple wildcards mismatch", "a*c*f", "abracadabra-e", false},
		{"no leading wildcard requires prefix match", "ab", "abc", false},
		{"no trailing wildcard requires suffix match", "bc", "abc", false},
		{"pattern longer than string", "abcd", "abc", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := wildcardMatch(tc.pattern, tc.s)
			require.Equal(t, tc.expected, result)
		})
	}
}
