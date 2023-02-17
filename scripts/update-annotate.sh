#!/bin/bash

kubectl create -n default configmap test-cm --from-literal=sample="test"

while :
do
    kubectl patch --field-manager=manager1 -n default configmap test-cm  --type json --patch '[{"op": "add", "path": "/data/sample", "value": "hoge"}]'
    sleep 1
    kubectl patch --field-manager=manager2 -n default configmap test-cm  --type json --patch '[{"op": "add", "path": "/data/sample", "value": "fuga"}]'
    sleep 1
done
