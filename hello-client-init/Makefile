VERSION = 0.1.0
ACCOUNT = freddygv
APP = hello-client-init

all: push-docker

build-docker:
	docker build -t $(APP):$(VERSION) .

push-docker: build-docker
    docker push $(REPO)/$(APP):$(VERSION)