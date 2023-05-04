#aws ecr get-login-password --region us-west-2 | docker login --username AWS --password-stdin 567662573502.dkr.ecr.us-west-2.amazonaws.com

#docker build -t eks-fargate-pod-mutating-hook .
#docker tag eks-fargate-pod-mutating-hook:latest 567662573502.dkr.ecr.us-west-2.amazonaws.com/eks-fargate-pod-mutating-hook:latest
#docker push 567662573502.dkr.ecr.us-west-2.amazonaws.com/eks-fargate-pod-mutating-hook:latest

aws ecr-public get-login-password --region us-east-1 | docker login --username AWS --password-stdin public.ecr.aws/q2l0r7l7
docker build -t fargate-mutating-webhook .
docker tag fargate-mutating-webhook:latest public.ecr.aws/q2l0r7l7/fargate-mutating-webhook:latest
docker push public.ecr.aws/q2l0r7l7/fargate-mutating-webhook:latest
