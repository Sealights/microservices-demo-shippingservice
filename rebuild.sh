#!/bin/bash

kubectl delete deployment b-shippingservice
kubectl delete svc b-shippingservice
kubectl create -f manifest.yaml