kubectl create ns fargate-scale-webhook
kubectl apply -f configs/certs.yaml
kubectl apply -f configs/rbac.yaml
kubectl apply -f configs/deployment.yaml
kubectl apply -f configs/service.yaml
kubectl apply -f configs/webhook.yaml

#kubectl create -f configs/example-pod.yaml
#kubectl create -f configs/nginx-hpa.yaml

#kubectl get pods --show-labels  -o wide | grep example-pod
#kubectl scale deployment/nginx --replicas=2 -n test
#kubectl run hazelcast --image=hazelcast --port=5701 -n test
