###############################################################################
#
# Makefile for project lifecycle
#
###############################################################################

# Default task: sets up development environment
install: up build

### PROJECT ###################################################################

# Builds the Go plugin
build: go-build restart-gateway

# Builds production-ready plugin bundle
bundle: go-bundle restart-gateway

# Outputs the project logs
logs: docker-logs

# Brings up the project
up: docker-up docker-status

# Brings down the project
down: docker-down docker-status

# Cleans the project
clean: go-clean

# Gets the status of the docker containers
status: docker-status

### DOCKER ####################################################################

# Gets the status of the running containers
.PHONY: docker-status
docker-status:
	docker-compose ps

# Gets the container logs
.PHONY: docker-logs
docker-logs:
	docker-compose logs -t --tail="all"

# Bring docker containers up
.PHONY: docker-up
docker-up:
	docker-compose up -d --remove-orphans tyk-dashboard

# Bring docker containers down
.PHONY: docker-down
docker-down:
	docker-compose down --remove-orphans

### Tyk Go Plugin ########################################################################

# Builds Go plugin and moves it into local Tyk instance
.PHONY: go-build
go-build:
	/bin/sh -c "cd ./go/src && go mod tidy && go mod vendor"
	docker-compose run --rm tyk-plugin-compiler CustomGoPlugin.so
	mv -f ./go/src/CustomGoPlugin_v4.1.0_linux_amd64.so ./tyk/middleware/CustomGoPlugin.so

# Builds production-ready Go plugin bundle as non-root user, using Tyk Bundler tool
.PHONY: go-bundle
go-bundle: go-build
	docker-compose run --rm --user=1000 --entrypoint "bundle/bundle-entrypoint.sh" tyk-gateway

# Cleans application files
.PHONY: go-clean
go-clean:
	-rm -rf ./go/src/vendor
	-rm -f ./tyk/middleware/CustomGoPlugin.so
	-rm -f ./tyk/bundle/CustomGoPlugin.so
	-rm -f ./tyk/bundle/bundle.zip

# Restarts the Tyk Gateway to instantly load new iterations of the Go plugin
.PHONY: restart-gateway
restart-gateway:
	-docker-compose restart tyk-gateway