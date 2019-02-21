REGISTRY?=mrferos
IMAGE?=newrelic-custom-metrics
VERSION?=latest

all: docker-push

test:
	go test -v ./...

docker-build: test
	docker build -t $(REGISTRY)/$(IMAGE):$(VERSION) .

docker-push: docker-build
	docker push $(REGISTRY)/$(IMAGE)
