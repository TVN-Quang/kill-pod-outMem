#!/bin/bash
#kubectl scale deployment nginx-deployment --replicas=3
kubectl label po $1 app-

kubectl delete po $1