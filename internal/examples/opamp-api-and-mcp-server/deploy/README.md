```
helm repo add open-telemetry https://open-telemetry.github.io/opentelemetry-helm-charts
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update

kubectl create namespace otel-agent
kubectl create namespace otel-gateway
kubectl create namespace prometheus

kubectl apply -f rbac.yaml

helm upgrade --install otel-gateway open-telemetry/opentelemetry-collector \
  --namespace otel-gateway \
  --create-namespace \
  -f otel-gateway-values.yaml

kubectl apply -f otel-gateway-metrics-svc.yaml

helm upgrade --install otel-agent open-telemetry/opentelemetry-collector \
  --namespace otel-agent \
  --create-namespace \
  -f otel-agent-values.yaml

kubectl apply -f otel-agent-metrics-svc.yaml

helm upgrade --install prometheus prometheus-community/prometheus \
  --namespace prometheus \
  --create-namespace \
  -f prometheus-values.yaml
```