VERSION = 0.1.0
ACCOUNT = freddygv
APP = hello-http

all: build

build:
	go build -o bin/hello

build-docker:
	docker build -t $(APP):$(VERSION) .

push-docker: build-docker
    docker push $(REPO)/$(APP):$(VERSION)