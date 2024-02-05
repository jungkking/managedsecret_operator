# managedsecret_operator

## Test Environment
- Ubuntu v22.04.3 LTS
- Go v1.21.6
- Kind v0.21.0
- Kubebuilder v3.14.0

## Test Commands
- kind create cluster
- kubectl create namespace dev
- kubectl create namespace test

- make docker-build IMG=test-image:0.1
- make generate
- make install
- make deploy IMG=test-image:0.1
- kind load docker-image test-image:0.1

- kubectl apply -k config/samples/
- kubectl get secrets --all-namespaces
- kubectl describe secret test-secret -n dev
- kubectl get secret test-secret -o jsonpath='{.data}' -n dev
- kubectl get secret test-secret -o jsonpath='{.data}' -n test
- kubectl delete managedsecret managedsecret-test1
- kubectl get secrets --all-namespaces