kubectl delete -f configs/example-pod.yaml
kubectl delete -f configs/webhook.yaml
kubectl delete -f configs/service.yaml
kubectl delete -f configs/deployment.yaml
kubectl delete -f configs/rbac.yaml
kubectl delete -f configs/certs.yaml
kubectl delete ns fargate-scale-webhook
