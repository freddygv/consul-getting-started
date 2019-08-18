VERSION = 0.1.0
ACCOUNT = freddygv
APP = hello-ttl

all: build

build:
	go build -o bin/hello

build-docker:
	docker build -t $(ACCOUNT)/$(APP):$(VERSION) .

push-docker: build-docker
	docker push $(ACCOUNT)/$(APP):$(VERSION)