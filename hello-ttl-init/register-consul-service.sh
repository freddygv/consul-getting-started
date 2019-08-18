#!/bin/sh

cat <<EOF > hello-service.json
{
  "Name": "hello",
  "Address": "${POD_IP}",
  "Port": 8080,
  "Check": {
    "CheckID": "hello-ttl",
    "Name": "5s TTL",
    "TTL": "5s"
  }
}
EOF

curl \
    --request PUT \
    --data @hello-service.json \
    "http://$HOST_IP:8500/v1/agent/service/register"