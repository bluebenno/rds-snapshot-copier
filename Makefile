GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
BINARY_NAME=rds-snapshot-copier
DOCKER=docker
DEP=dep

all: dep test build

build:
		cd cmd/rds-snapshot-copier && $(GOBUILD) -o ../../$(BINARY_NAME) -v
test:
		$(GOTEST) -v ./...
clean:
		$(GOCLEAN)
		rm -f $(BINARY_NAME)
		rm -f $(BINARY_UNIX)
dep:
		$(DEP) ensure

docker:
		$(DOCKER) build -t $(BINARY_NAME) .
