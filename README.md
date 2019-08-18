# Lab: Getting started with Consul on K8s

#### Prerequisites
- minikube (if running locally)
- kubectl
- helm

#### Instructions
##### Setup
1. `minikube start --memory 8192` (if running locally)
2. `helm init`
3. `make deps`
4. `helm install -f helm-consul-values.yaml --name hedgehog ./consul-helm`
5. `./configure-dns.sh`
6. `kubectl create -f deployments/`
7. `minikube dashboard`
8. `minikube service hedgehog-consul-ui`

##### Dynamic Configuration
Get the list of pods and find one that is running a Consul agent. 
We'll use this as an easy way to run Consul CLI commands.
`$ kubectl get pods`

Look for one with consul in the name and connect to the running pod.
`$ kubectl exec -it giggly-echidna-consul-5t2dc /bin/sh`

Once connected, run a command that saves a value to Consul.
`$ consul kv put service/hello/hello-http/enable_checks false`

Switch to the Consul UI and note that the HTTP check for the hello service is failing.

##### Teardown
`minikube delete` (if using minikube)