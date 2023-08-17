#!/bin/bash

istioctl kube-inject -f redis_exporter.yaml | kubectl apply -f -
istioctl kube-inject -f mysql_exporter.yaml | kubectl apply -f -

status=$(kubectl get pod -n hipster  |grep mysql | awk '{print $3}')
ready="Running"

while [ "$status" != "$ready" ]
do
    echo "Mysql is not ready!"
    sleep 30
    status=$(kubectl get pod -n hipster  |grep mysql | awk '{print $3}')
done

echo "Mysql is ready now!"

istioctl kube-inject -f adservice.yaml | kubectl apply -f -
istioctl kube-inject -f checkoutservice.yaml | kubectl apply -f -
istioctl kube-inject -f emailservice.yaml | kubectl apply -f -
istioctl kube-inject -f paymentservice.yaml  | kubectl apply -f -
istioctl kube-inject -f recommendationservice.yaml | kubectl apply -f -
istioctl kube-inject -f shippingservice.yaml | kubectl apply -f -
istioctl kube-inject -f cartservice.yaml | kubectl apply -f -
istioctl kube-inject -f currencyservice.yaml | kubectl apply -f -
istioctl kube-inject -f frontend.yaml | kubectl apply -f -
istioctl kube-inject -f productcatalogservice.yaml | kubectl apply -f -
