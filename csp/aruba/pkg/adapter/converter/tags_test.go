package converter_test

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"

	res "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	instancedom "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/instance"
	netdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network"
	publicipdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/public-ip"
	securitygroup "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group"
	securitygrouprule "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group-rule"
	subnetdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/subnet"
	bsdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage"
	wsdom "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1"

	"github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/adapter/converter"
)

var testLabels = map[string]string{"env": "prod", "app": "web"}

// sorted rendering of testLabels
var testTags = []string{"app-web", "env-prod"}

func TestArubaTags(t *testing.T) {
	require.Nil(t, converter.ArubaTags(nil))
	require.Nil(t, converter.ArubaTags(map[string]string{}))
	require.Equal(t, testTags, converter.ArubaTags(testLabels))
}

// The Aruba CMP accepts only letters, digits and "-" in a tag and fails the whole resource
// otherwise, so anything else has to be replaced before it is sent - in the value as much as the
// key, since Kubernetes label values routinely contain "." and "_".
func TestArubaTags_staysInsideTheAlphabetArubaAccepts(t *testing.T) {
	valid := regexp.MustCompile(`^[A-Za-z0-9-]+$`)

	for name, labels := range map[string]map[string]string{
		"dotted value":      {"version": "24.04"},
		"underscored value": {"os_name": "ubuntu_24"},
		"slash and colon":   {"app.kubernetes.io/name": "team:platform"},
		"spaces":            {"owner": "platform team"},
		"unicode":           {"région": "münchen"},
		"already clean":     {"cost-center": "eng-42"},
	} {
		t.Run(name, func(t *testing.T) {
			for _, tag := range converter.ArubaTags(labels) {
				require.Truef(t, valid.MatchString(tag), "tag %q would be rejected by Aruba", tag)
			}
		})
	}
}

func TestArubaTags_rendering(t *testing.T) {
	// A single dash already in a key or value survives; a run of rejected characters collapses.
	require.Equal(t, []string{"cost-center-eng-42"}, converter.ArubaTags(map[string]string{"cost-center": "eng-42"}))
	require.Equal(t, []string{"version-24-04"}, converter.ArubaTags(map[string]string{"version": "24.04"}))

	// No trailing dash when the value is empty or unusable, and no leading dash from the key.
	require.Equal(t, []string{"empty"}, converter.ArubaTags(map[string]string{"empty": ""}))
	require.Equal(t, []string{"key"}, converter.ArubaTags(map[string]string{"key": "..."}))

	// A label with nothing representable is dropped rather than sent as an empty tag.
	require.Nil(t, converter.ArubaTags(map[string]string{"...": "..."}))

	// Sanitising can map two distinct labels onto the same tag; Aruba should not be sent the
	// duplicate.
	require.Equal(t, []string{"a-b"}, converter.ArubaTags(map[string]string{"a": "b", "a.": ".b"}))
}

func regionalMeta(name string) commondomain.RegionalMetadata {
	return commondomain.RegionalMetadata{
		CommonMetadata: commondomain.CommonMetadata{Name: name},
		Scope:          res.Scope{Tenant: "test-tenant", Workspace: "test-workspace"},
		Labels:         testLabels,
	}
}

// Every Aruba resource the plugin creates must carry the SECA labels of the resource it was
// converted from - they are the only place a user's labels land on the Aruba side.
func TestSECALabelsBecomeArubaTags(t *testing.T) {
	t.Run("network", func(t *testing.T) {
		vpc, err := converter.NewNetworkVPCConverter().FromSECAToAruba(&netdom.Network{
			RegionalMetadata: regionalMeta("net-1"),
		})
		require.NoError(t, err)
		require.Equal(t, testTags, vpc.Spec.Tags)
	})

	t.Run("subnet", func(t *testing.T) {
		sub, err := converter.NewSubnetConverter().FromSECAToAruba(&subnetdom.Subnet{
			RegionalNetworkMetadata: commondomain.RegionalNetworkMetadata{
				RegionalMetadata: regionalMeta("sub-1"),
				Network:          "net-1",
			},
			Spec: subnetdom.SubnetSpec{Cidr: subnetdom.CIDR{IPv4: "10.0.0.0/24"}},
		})
		require.NoError(t, err)
		require.Equal(t, testTags, sub.Spec.Tags)
	})

	t.Run("block storage", func(t *testing.T) {
		bs, err := converter.NewBlockStorageConverter().FromSECAToAruba(&bsdom.BlockStorage{
			RegionalMetadata: regionalMeta("vol-1"),
			Spec:             bsdom.BlockStorageSpec{SizeGB: 20},
		})
		require.NoError(t, err)
		require.Equal(t, testTags, bs.Spec.Tags)
	})

	t.Run("public ip", func(t *testing.T) {
		eip, err := converter.NewPublicIpElasticIpConverter().FromSECAToAruba(&publicipdom.PublicIp{
			RegionalMetadata: regionalMeta("ip-1"),
		})
		require.NoError(t, err)
		require.Equal(t, testTags, eip.Spec.Tags)
	})

	t.Run("workspace", func(t *testing.T) {
		prj, err := converter.NewWorkspaceProjectConverter().FromSECAToAruba(&wsdom.Workspace{
			RegionalMetadata: regionalMeta("test-workspace"),
			Spec:             wsdom.WorkspaceSpec{},
		})
		require.NoError(t, err)
		require.Equal(t, testTags, prj.Spec.Tags)
	})

	t.Run("instance and its key pair", func(t *testing.T) {
		inst := &instancedom.Instance{RegionalMetadata: regionalMeta("vm-1")}
		require.Equal(t, testTags, converter.BuildCloudServer(inst, converter.CloudServerRefs{}).Spec.Tags)
		require.Equal(t, testTags, converter.BuildKeyPair(inst, "ssh-rsa AAAAKEY").Spec.Tags)
	})

	t.Run("security group and its rules", func(t *testing.T) {
		sg := converter.BuildSecurityGroup("web", "net-1", "", "test-tenant", "ws-ns", testLabels, vpcRef(), projectRef())
		require.Equal(t, testTags, sg.Spec.Tags)

		// An inline rule has no labels of its own and inherits the group's; a standalone
		// SecurityGroupRule is its own SECA resource and keeps its own.
		rules := converter.NormalizeInlineRules(
			[]securitygroup.SecurityGroupRuleSpec{{Direction: "ingress", Protocol: "tcp"}},
			testLabels,
		)
		rules = append(rules, converter.NormalizeStandaloneRule(
			securitygrouprule.SecurityGroupRuleSpec{Direction: "egress", Protocol: "udp"},
			map[string]string{"tier": "db"},
		))

		out := converter.BuildSecurityRules(rules, sg.Name, "", "test-tenant", "ws-ns", vpcRef(), projectRef())
		require.Len(t, out, 2)
		require.Equal(t, testTags, out[0].Spec.Tags)
		require.Equal(t, []string{"tier-db"}, out[1].Spec.Tags)
	})
}
