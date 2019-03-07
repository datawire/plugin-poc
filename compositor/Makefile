DOCKER_REGISTRY ?= localhost:31000
DOCKER_IMAGE = $(DOCKER_REGISTRY)/compositor:$(shell git describe --tags --always --dirty)

all: .docker.stamp
.PHONY: all

.docker.stamp: $(wildcard *.go) go.mod Dockerfile
	docker build -t $(DOCKER_IMAGE) .
	date > $@

push: .docker.stamp
	docker push $(DOCKER_IMAGE)
.PHONY: push

clean:
	rm -f -- .docker.stamp
.PHONY: clean
