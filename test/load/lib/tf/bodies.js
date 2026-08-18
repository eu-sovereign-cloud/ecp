// Minimal resource bodies (aligned with test/e2e/flow_test.go).

export function workspaceBody() {
  return {};
}

export function networkBody(cidrV4) {
  return {
    spec: {
      cidr: { ipv4: cidrV4 },
      skuRef: { resource: 'sku-1' },
    },
  };
}

export function routeTableBody(networkCidr) {
  return {
    spec: {
      routes: [
        {
          destinationCidrBlock: networkCidr,
          targetRef: { resource: 'internet-gateways/igw' },
        },
      ],
    },
  };
}

export function subnetBody(cidrV4, routeTableName, zone) {
  return {
    spec: {
      cidr: { ipv4: cidrV4 },
      routeTableRef: { resource: `route-tables/${routeTableName}` },
      zone,
    },
  };
}

export function blockStorageBody(sizeGB) {
  return {
    spec: {
      sizeGB,
      skuRef: { resource: 'sku-1' },
    },
  };
}

export function instanceBody(blockStorageName, zone) {
  return {
    spec: {
      bootVolume: {
        deviceRef: { resource: `block-storages/${blockStorageName}` },
      },
      skuRef: { resource: 'sku-1' },
      zone,
    },
  };
}

export function nicBody(subnetName) {
  return {
    spec: {
      addresses: ['0.0.0.0'],
      subnetRef: { resource: `subnets/${subnetName}` },
    },
  };
}
