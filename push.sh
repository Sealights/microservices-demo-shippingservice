#!/bin/bash

docker build -t microservices-demo-shippingservice .
docker tag microservices-demo-shippingservice:latest 159616352881.dkr.ecr.eu-west-1.amazonaws.com/microservices-demo-shippingservice:latest
docker push 159616352881.dkr.ecr.eu-west-1.amazonaws.com/microservices-demo-shippingservice:latest