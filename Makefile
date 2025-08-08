KIND_CLUSTER_NAME=crossplanetest
KIND_VERSION=v0.29.0
CROSSPLANE_NAMESPACE=crossplane-system

crossplane-local-dev: ensure-kind ensure-helm kind-create crossplane-install ionos-crossplane-provider-install

ensure-kind:
	@command -v kind >/dev/null 2>&1 || { \
		echo "kind not found, installing..."; \
		curl -Lo ./kind https://kind.sigs.k8s.io/dl/$(KIND_VERSION)/kind-linux-amd64; \
		chmod +x ./kind; \
		sudo mv ./kind /usr/local/bin/kind; \
	}

ensure-helm:
	@command -v helm >/dev/null 2>&1 || { \
		echo "helm not found, installing..."; \
		curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash; \
	}

kind-create:
	kind create cluster --name $(KIND_CLUSTER_NAME) || true

crossplane-install:
	helm repo add crossplane-stable https://charts.crossplane.io/stable
	helm repo update
	helm install crossplane --namespace $(CROSSPLANE_NAMESPACE) --create-namespace crossplane-stable/crossplane

run-server:
	go run ./main.go apiserver

ionos-crossplane-provider-install:
	kubectl apply -f https://raw.githubusercontent.com/ionos-cloud/provider-upjet-ionoscloud/refs/heads/main/package/crds/compute.ionoscloud.io_datacenters.yaml
	kubectl apply -f https://raw.githubusercontent.com/ionos-cloud/provider-upjet-ionoscloud/refs/heads/main/package/crds/compute.ionoscloud.io_servers.yaml
	kubectl apply -f https://raw.githubusercontent.com/ionos-cloud/provider-upjet-ionoscloud/refs/heads/main/package/crds/upjet-ionoscloud.ionoscloud.io_providerconfigs.yaml
	kubectl apply -f https://raw.githubusercontent.com/ionos-cloud/provider-upjet-ionoscloud/refs/heads/main/package/crds/upjet-ionoscloud.ionoscloud.io_providerconfigusages.yaml
	kubectl apply -f https://raw.githubusercontent.com/crossplane/crossplane/release-1.15/cluster/crds/pkg.crossplane.io_providers.yaml
	kubectl create secret generic --namespace $(CROSSPLANE_NAMESPACE) example-provider-secret --from-literal=credentials="{\"token\":\"${IONOS_TOKEN}\"}" --dry-run=client -o yaml | kubectl apply -f -
	kubectl apply -f https://raw.githubusercontent.com/ionos-cloud/provider-upjet-ionoscloud/refs/heads/main/examples/providerconfig/providerconfig.yaml
	kubectl apply -f https://raw.githubusercontent.com/ionos-cloud/provider-upjet-ionoscloud/refs/heads/main/examples/install.yaml -n $(CROSSPLANE_NAMESPACE)

clean-crossplane-dev:
	kubectl delete namespace $(CROSSPLANE_NAMESPACE) || true
	kubectl delete secret example-provider-secret --namespace $(CROSSPLANE_NAMESPACE) || true
	kind delete cluster --name $(KIND_CLUSTER_NAME) || true
