REGISTRY?=mrferos
IMAGE?=newrelic-custom-metrics
VERSION?=latest

install:
	glide install -v

test: install
	go test -v ./...

docker-build: test
	docker build -t $(REGISTRY)/$(IMAGE):$(VERSION) .

docker-push: docker-build
	docker push $(REGISTRY)/$(IMAGE)
