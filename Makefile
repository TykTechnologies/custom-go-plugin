###############################################################################
#
# Makefile for project lifecycle
#
###############################################################################

# Set TYK environment variables
export TYK_VERSION := v5.0.1
export ARCH := amd64
export OS := linux

# Default task: sets up development environment
install: up build

### PROJECT ###################################################################

# Builds the Go plugin
build: go-build restart-gateway

# Builds production-ready plugin bundle
bundle: go-bundle restart-gateway

# Outputs the project logs
logs: docker-logs

# Outputs the gateway log with formatting to make it easier to read in local dev
log: docker-gateway-log

# Brings up the project
up: docker-up bootstrap docker-status

# Brings down the project
down: docker-down docker-status

# Cleans the project
clean: docker-clean go-clean

# Gets the status of the docker containers
status: docker-status

up-oss: docker-up-oss bootstrap-oss docker-status

### DOCKER ####################################################################

# Gets the status of the running containers
.PHONY: docker-status
docker-status:
	docker-compose ps

# Gets the container logs
.PHONY: docker-logs
docker-logs:
	docker-compose logs -t --tail="all"

# Gets the container log for gateway and applies formatting for easier reading in local dev
.PHONY: docker-gateway-log
docker-gateway-log:
	docker-compose logs tyk-gateway -t -f | perl -ne 'if (/time="([^"]+)" level=(\w+) msg="((?:\\"|[^"])*)"(\s*prefix=([^\s]+))?/) { print "$$1 ".sprintf("%-20s", "[$$2]".($$5 ? "[".substr($$5,0,10)."] " : (" " x 12)))."$$3\n" }'

# Bring docker containers up
.PHONY: docker-up
docker-up:
	docker-compose up -d --remove-orphans tyk-dashboard

# Bootstrap dashboard
.PHONY: bootstrap
bootstrap:
	$(shell ./tyk/scripts/bootstrap.sh)

# Bring docker containers down
.PHONY: docker-down
docker-down:
	docker-compose down --remove-orphans

# Clean docker containers volumes
.PHONY: docker-clean
docker-clean:
	docker-compose down --volumes --remove-orphans

### Tyk Go Plugin ########################################################################

go/src/go.mod:
	cd ./go/src ; \
	go mod init tyk-plugin ; \
	go get -d github.com/TykTechnologies/tyk@`git ls-remote https://github.com/TykTechnologies/tyk.git refs/tags/${TYK_VERSION} | awk '{print $$1;}'` ; \
	go mod tidy ; \
	go mod vendor

# Builds Go plugin and moves it into local Tyk instance
.PHONY: go-build
go-build: go/src/go.mod
	/bin/sh -c "cd ./go/src && go mod tidy && go mod vendor"
	docker-compose run --rm tyk-plugin-compiler CustomGoPlugin.so _$$(date +%s)
	mv -f ./go/src/CustomGoPlugin*.so ./tyk/middleware/

# Runs Go Linter
lint:
	/bin/sh -c "docker run --rm -v ${PWD}/go/src:/app -v ~/.cache/golangci-lint/v1.53.2:/root/.cache -w /app golangci/golangci-lint:v1.53.2 golangci-lint run"

# Runs Go unit tests
test:
	/bin/sh -c "cd ./go/src && go test"

# Run Go test coverage
coverage:
	mkdir -p /tmp/test-results ; \
	cd ./go/src ; \
	go test ./... -coverprofile coverage.out -covermode count ; \
	grep -v tyk-plugin/tyk_util.go coverage.out > coverage.out.tmp ; \
	mv coverage.out.tmp coverage.out ; \
	go tool cover -func coverage.out ; \
	go tool cover -html=coverage.out -o coverage.html ; \
	mv coverage.out coverage.html /tmp/test-results ; \
	totalCoverage=`go tool cover -func=/tmp/test-results/coverage.out | grep total | grep -Eo '[0-9]+\.[0-9]+'` ; \
	echo "Total Coverage: $$totalCoverage %" ; \
	rm -rf /tmp/test-results

# Builds production-ready Go plugin bundle as non-root user, using Tyk Bundler tool
.PHONY: go-bundle
go-bundle: go-build
	docker-compose run --rm --user=1000 --entrypoint "bundle/bundle-entrypoint.sh" tyk-gateway

# Cleans application files
.PHONY: go-clean
go-clean:
	-rm -rf ./go/src/vendor
	-rm -rf ./go/src/go.mod
	-rm -rf ./go/src/go.sum
	-rm -f ./tyk/middleware/CustomGoPlugin*.so
	-rm -f ./tyk/bundle/CustomGoPlugin.so
	-rm -f ./tyk/bundle/bundle.zip

# Restarts the Tyk Gateway to instantly load new iterations of the Go plugin
.PHONY: restart-gateway
restart-gateway:
	docker-compose restart tyk-gateway

.PHONY: docker-up-oss
docker-up-oss:
	docker-compose -f docker-compose-oss.yml up -d

# Bootstrap dashboard
.PHONY: bootstrap-oss
bootstrap-oss:
	$(shell ./tyk/scripts/bootstrap-oss.sh)
