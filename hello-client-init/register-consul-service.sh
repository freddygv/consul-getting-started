#!/bin/sh

cat <<EOF > hello-client-service.json
{
  "Name": "client"
}
EOF

curl \
    --request PUT \
    --data @hello-client-service.json \
    "http://$HOST_IP:8500/v1/agent/service/register"