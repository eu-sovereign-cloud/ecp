# Aruba Network and Compute Resources Mapping

This document describes the mapping implemented between SECA network and compute resources and Aruba resources. The mapping is not 1:1: Aruba models fewer objects than SECA does, so some SECA resources have no Aruba counterpart at all, and some Aruba resources (a key pair) have no SECA counterpart and are synthesised.

The Aruba resources are the custom resources of the [arubacloud-resource-operator](https://github.com/Arubacloud/arubacloud-resource-operator) `v1.1.1` (`arubacloud.com/v1alpha1`), which the plugin creates and then watches; the operator is what actually talks to the Aruba CMP.

## Mapping

| SECA Resource         | Aruba Resource      | Waits for                                                       |
|-----------------------|---------------------|-----------------------------------------------------------------|
| Network               | `VPC`               | Workspace active, an `InternetGateway` in the workspace, `Project` active |
| Subnet                | `Subnet`            | Workspace active, `Project` active, parent `VPC` active          |
| Public IP             | `ElasticIP`         | Workspace active, `Project` active                               |
| Route Table           | *none*              | —                                                               |
| Internet Gateway      | *none*              | —                                                               |
| Security Group        | *none directly* — materialised as `SecurityGroup` at instance attach | — |
| Security Group Rule   | *none directly* — materialised as `SecurityRule` at instance attach | — |
| NIC                   | *none*              | Aruba has no standalone NIC: NICs are attributes of a `CloudServer` |
| Network SKU           | not reconciled      | Read-only catalog                                                |
| Instance              | `CloudServer` (+ `KeyPair`, `SecurityGroup`, `SecurityRule`) | Workspace & `Project` active, its NICs present, their subnets active, an ssh key, a boot volume |
| Compute SKU           | not reconciled      | Read-only catalog; the sku name is the CloudServer `FlavorName`  |

### Route Table and Internet Gateway have no Aruba counterpart

Aruba's network API (see the [SDK](https://github.com/Arubacloud/sdk-go) `client_network.go`) exposes VPCs, subnets, security groups and rules, elastic IPs, VPC peerings and VPN tunnels — there is no route table and no internet gateway object. Internet egress and intra-VPC routing are properties Aruba configures on the VPC itself.

Both SECA resources are therefore accepted and go active immediately without creating anything. They exist so the SECA model stays consistent across providers. The `InternetGateway` additionally gates the Network: an Aruba VPC always provides internet egress, so the SECA resource representing that egress must exist before the VPC is created. **The Route Table gates nothing** — see below.

### Why the Route Table is not a precondition

The Aruba API can auto-create a default subnet and security group together with a VPC, via the `preset` boolean in the create request (`VPCPropertiesInnerRequest.Preset` in the SDK). The operator hard-codes `NotDefault().WithoutPreset()` and its `VPCSpec` does not expose the field, so `preset` is always `false`: creating a VPC never derives a subnet or any routing object. Subnets are created independently afterwards, as their own custom resources. Because nothing is derived from the VPC, there is nothing for a Route Table to be a precondition of.

### Security groups are materialised at instance attach, not at creation

Aruba's `SecurityGroup` and `SecurityRule` both require a `VPCReference` (a rule additionally requires a `SecurityGroupReference`). A SECA `SecurityGroup` carries **no VPC**: it is workspace-scoped and only takes effect once a NIC or instance references it (via `securityGroupRefs`). That reference is what binds the group to a subnet, hence to a network, hence to an Aruba VPC. So the SECA `SecurityGroup` and `SecurityGroupRule` controllers create nothing — they accept and go active — and the real Aruba `SecurityGroup` + `SecurityRule` are created by the **compute-instance handler**, per VPC, when an instance attaches the group (see "Compute" below). A standalone SECA `SecurityGroupRule` is a reusable template that does nothing until a `SecurityGroup` pulls it in via `ruleRefs`; it is materialised as part of the group that references it.

Replicating a SECA security group into *every* VPC up front was considered and rejected: it would create dormant Aruba resources nothing uses, and would need the Network handler to back-fill security groups into VPCs created later, for no present benefit.

## Compute

A SECA `Instance` maps to an Aruba `CloudServer`. A CloudServer's required references — VPC, subnets, security groups, key pair, boot volume — are **none of them named directly** by the Instance; the handler resolves them from the instance's dependency graph and materialises what Aruba needs but SECA does not model as its own resource:

- **NICs → subnets → VPC.** The instance's `primaryNicRef`/`additionalNicRefs` are loaded; each NIC's `subnetRef`, `securityGroupRefs` and `publicIpRefs` are collected. A NIC reference carries no network, so the Aruba `Subnet` backing a SECA subnet name is located by workspace label across namespaces and matched by name; the first active match wins. The subnet's `VPCReference` fixes the CloudServer's VPC.
- **Security groups + rules** referenced by the NICs (and the instance's own `securityGroupRef`) are materialised in that VPC: one Aruba `SecurityGroup` per `(network, SECA group)`, named `<seca-group>-<network>`, plus its `SecurityRule`s built from the group's inline `rules` and its `ruleRefs`.
- **Key pair.** An Aruba CloudServer requires a `KeyPairReference`, but SECA has no key-pair resource: the public key travels inline in `Instance.Spec.SshKeys` (the field is documented as "references" but actually carries the key material, e.g. `ssh-rsa AAAA…`). The handler creates a `KeyPair` named `<instance>-key` from the first ssh key and deletes it with the instance.
- **Boot/data volumes** come from `BootVolume`/`DataVolumes` (Aruba `BlockStorage`, already reconciled by the storage plugin); **flavor** is the last path segment of `SkuRef`; an optional **elastic IP** comes from the first NIC public IP.

Missing dependencies gate the create with `ErrStillProcessing` (the instance stays in `creating` and is retried): a NIC not created yet, a subnet not yet active, **no security group** (a CloudServer requires ≥1), or **no ssh key** (the required `KeyPairReference` cannot be built). `PowerOn`/`PowerOff` are **no-ops**: Aruba's `CloudServer` CRD exposes no power field, so power state lives only on the SECA side.

On instance delete the CloudServer is deleted and then the key pair; the **materialised security groups are left in place** — they may be shared with other instances and do not interfere once the SECA security group still exists.

### Compute mapping ceilings

| Behaviour | Detail |
|---|---|
| Multiple ssh keys | Aruba `KeyPair` holds a single value; SECA allows up to 32 → the first is used. |
| Security-group rule changes | Rules are created once, at attach; later edits to the SECA group are not reconciled onto the Aruba side. |
| Multi-VPC instance | All of an instance's subnets are assumed to share one network/VPC; the first subnet's VPC wins. |
| Rule fan-out | One SECA rule expands to one Aruba `SecurityRule` per `(protocol × port × source)`: `tcp+udp` → TCP and UDP rules, a port list → one rule per port, each `sourceRef` → its own rule. |
| Rule targets | A `security-groups/<name>` source becomes a `SecurityGroup` target; anything else is treated as an IP/CIDR literal (`Ip` target). No source means all traffic → `0.0.0.0/0`. Instance/gateway sources have no Aruba target type and are mapped best-effort to `Ip`. |

## What is not propagated

| SECA field | Why |
|---|---|
| `RouteTable.Spec.Routes` | Aruba has no route table. The nearest primitive is per-subnet DHCP static routes, which the operator's `Subnet` CRD does not expose (only `dhcp.enabled`). The SECA route-table plugin interface also has no `Update`, so route edits never reach the plugin. |
| `Network.Spec.CIDR`, `AdditionalCIDRs` | An Aruba `VPC` has no address range; addressing is defined per subnet. |
| `Network.Spec.SkuRef`, `Subnet.Spec.SkuRef` | No SKU concept on the Aruba side for these. |
| `Subnet.Spec.Zone`, `Subnet.Spec.RouteTableRef` | No equivalent field on the Aruba `Subnet` (see routes, above). |
| `InternetGateway.Spec.EgressOnly` | No gateway object to configure. |
| `Instance.Spec.UserData`, `AntiAffinityGroup` | No equivalent field on the Aruba `CloudServer`. |
| `Instance.Spec.PrimaryNicRef`/`AdditionalNicRefs` | Not propagated as NICs (Aruba has none); consumed to resolve the CloudServer's subnets and security groups. |

Some SECA specs are **rejected** rather than silently ignored, because honouring them partially would hand back something the user did not ask for:

- `PublicIp.Spec.Address` (BYOIP) — an Aruba `ElasticIP` has no address field; Aruba always allocates it.
- `PublicIp.Spec.Version: IPv6` — `ElasticIP` is IPv4-only.

An IPv6-only `Subnet` is likewise rejected: the Aruba `Subnet` CRD validates `cidr` against an IPv4 pattern. Dual-stack subnets are accepted, with the IPv6 range dropped.

## Defaults applied

| Field | Value | Why |
|---|---|---|
| Region | `ITBG-Bergamo` | When the SECA resource carries no region. |
| `Subnet.Spec.Type` | `Advanced` | Lets the subnet use the CIDR from the SECA spec; `Basic` would have Aruba choose the range and silently ignore it. |
| `Subnet.Spec.DHCP.Enabled` | `true` | The CRD requires the field; SECA has no knob for it. |
| `ElasticIP.Spec.BillingPeriod` | `Hour` | The CRD requires the field; SECA has no knob for it. |
| `CloudServer.Spec.Zone` | `ITBG-1` | When the SECA `Instance` carries no zone. |

## Status reporting limitation

While a resource waits for a dependency, the plugin returns `ErrStillProcessing`, which the framework turns into a plain requeue. The SECA resource stays in state `creating` with **no message naming the missing dependency** — the plugin interface has no channel for a message on an in-progress wait (returning a real error would surface a condition, but flip the state to `error`).

So a Network with no Internet Gateway, or an Instance with no ssh key, sits in `creating` indefinitely rather than reporting the reason. Surfacing that needs a framework change to the plugin contract.

## Namespaces and references

Aruba CRs are created in the same namespace as the SECA resource they mirror (hashed by scope, via `ComputeNamespace`/`ComputeNetworkNamespace` in `framework/backend/kubernetes/adapter.go`):

- `VPC`, `ElasticIP`, `CloudServer`, `KeyPair` and the materialised `SecurityGroup`/`SecurityRule` → workspace namespace, `hash(tenant/workspace)`.
- `Subnet` → its network's namespace, `hash(tenant/workspace/network)`.
- `Project` (from the SECA Workspace) → tenant namespace, `hash(tenant)`.

Every Aruba CR references its `Project` by `{name: <workspace>, namespace: hash(tenant)}`; the Aruba `Subnet`, `SecurityGroup`, `SecurityRule` and `CloudServer` additionally reference their `VPC` by `{name: <seca network>, namespace: hash(tenant/workspace)}`. The `CloudServer` references subnets in the network namespace and its key pair, security groups and volumes in the workspace namespace — cross-namespace references the operator supports.
