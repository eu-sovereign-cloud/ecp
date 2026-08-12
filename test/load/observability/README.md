# Load-test observability

Grafana dashboards for the ECP gateway load-test stack (Prometheus job
`ecp-gateway` plus cluster cAdvisor / API server scrapes).

## Dashboards

| File | UID | Purpose |
|------|-----|---------|
| `dashboards/ecp-gateway.json` | `ecp-gateway` | Primary auth latency (middleware / authz / RBAC) and light runtime |
| `dashboards/ecp-gateway-loadtest.json` | `ecp-gateway-loadtest` | Second dashboard for load runs: request counts, memory, GC, upstream K8s |

The load-test board uses **histogram `_count`** for request rate and totals
(`ecp_gateway_auth_middleware_duration_seconds_count`). There is no separate
request counter in the gateway.

## Apply to the monitoring namespace

```bash
./apply-dashboards.sh
# or: make -C test/load apply-dashboards
```

Defaults: `NAMESPACE=monitoring`, `CONFIGMAP=grafana-dashboards`.

Open Grafana (folder **ECP**):

- [ECP Gateway](/d/ecp-gateway)
- [ECP Gateway Load Test](/d/ecp-gateway-loadtest)
