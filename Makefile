default: all

# Do a parallel build with multiple jobs, based on the number of CPUs online
# in this system: 'make -j8' on a 8-CPU system, etc.
ifeq ($(JOBS),)
  JOBS := $(shell grep -c ^processor /proc/cpuinfo 2>/dev/null)
  ifeq ($(JOBS),)
    JOBS := $(shell sysctl -n hw.logicalcpu 2>/dev/null)
    ifeq ($(JOBS),)
      JOBS := 1
    endif
  endif
endif

use_all_cores:
	make -j$(JOBS) all

BUILDER := covenantsql/covenantsql-builder
IMAGE := covenantsql/covenantsql

GIT_COMMIT ?= $(shell git rev-parse --short HEAD)
GIT_DIRTY ?= $(shell test -n "`git status --porcelain`" && echo "+CHANGES" || true)
GIT_DESCRIBE ?= $(shell git describe --tags --always)

COMMIT := $(GIT_COMMIT)$(GIT_DIRTY)
VERSION := $(GIT_DESCRIBE)
SHIP_VERSION := $(shell docker image inspect -f "{{ .Config.Labels.version }}" $(IMAGE):latest 2>/dev/null)
IMAGE_TAR := $(subst /,_,$(IMAGE)).$(SHIP_VERSION).tar
IMAGE_TAR_GZ := $(IMAGE_TAR).gz

status:
	@echo "Commit: $(COMMIT) Version: $(VERSION) Ship Version: $(SHIP_VERSION)"


builder: status
	# alpine image libmusl is not compatible with golang race detector
	# also alpine libmusl is required for building static binaries to avoid glibc getaddrinfo panic
	docker build \
		--tag $(BUILDER):$(VERSION) \
		--tag $(BUILDER):latest \
		--build-arg BUILD_ARG=release \
		-f docker/builder.Dockerfile \
		.

docker: builder
	docker build \
		--tag $(IMAGE):$(VERSION) \
		--tag $(IMAGE):latest \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg VERSION=$(VERSION) \
		-f docker/Dockerfile \
		.

docker_clean: status
	docker rmi -f $(BUILDER):latest
	docker rmi -f $(IMAGE):latest
	docker rmi -f $(BUILDER):$(VERSION)
	docker rmi -f $(IMAGE):$(VERSION)

alpine_release: builder
	temp_container=$$(docker create $(BUILDER):$(VERSION)) ; \
	docker cp $${temp_container}:/go/src/github.com/CovenantSQL/CovenantSQL/bin - | gzip > app-bin.tgz

save: status
ifeq ($(SHIP_VERSION),)
	$(error No version to ship, please build first)
endif
	docker save $(IMAGE):$(SHIP_VERSION) > $(IMAGE_TAR)
	tar zcf $(IMAGE_TAR_GZ) $(IMAGE_TAR)

start:
	docker-compose down
	docker-compose up --no-start
	docker-compose start

stop:
	docker-compose down

logs:
	docker-compose logs -f --tail=10

push_testnet:
	docker tag $(IMAGE):$(VERSION) $(IMAGE):testnet
	docker push $(IMAGE):testnet

push_bench:
	docker tag $(IMAGE):$(VERSION) $(IMAGE):bench
	docker push $(IMAGE):bench

push_staging:
	docker tag $(IMAGE):$(VERSION) $(IMAGE):staging
	docker push $(IMAGE):staging


push:
	docker push $(IMAGE):$(VERSION)
	docker push $(IMAGE):latest



branch := $(shell git rev-parse --abbrev-ref HEAD)
builddate := $(shell date +%Y%m%d%H%M%S)

unamestr := $(shell uname)

ifeq ($(unamestr),Linux)
	platform := linux
else
  ifeq ($(unamestr),Darwin)
	platform := darwin
  endif
endif

ifdef CQLVERSION
	version := $(CQLVERSION)-$(builddate)
else
	version := $(branch)-$(GIT_COMMIT)-$(builddate)
endif

tags := $(platform) sqlite_omit_load_extension
miner_tags := $(tags) sqlite_vtable sqlite_fts5 sqlite_icu sqlite_json
test_tags := $(tags) testbinary
miner_test_tags := $(test_tags) sqlite_vtable sqlite_fts5 sqlite_icu sqlite_json

static_flags := -linkmode external -extldflags '-static'
test_flags := -coverpkg github.com/CovenantSQL/CovenantSQL/... -cover -race -c

ldflags_role_bp := -X main.version=$(version) -X github.com/CovenantSQL/CovenantSQL/conf.RoleTag=B
ldflags_role_miner := -X main.version=$(version) -X github.com/CovenantSQL/CovenantSQL/conf.RoleTag=M
ldflags_role_client := -X main.version=$(version) -X github.com/CovenantSQL/CovenantSQL/conf.RoleTag=C
ldflags_role_client_simple_log := $(ldflags_role_client) -X github.com/CovenantSQL/CovenantSQL/utils/log.SimpleLog=Y

GOTEST := CGO_ENABLED=1 go test $(test_flags) -tags "$(test_tags)"
GOTEST_MINER := CGO_ENABLED=1 go test $(test_flags) -tags "$(miner_test_tags)"
GOBUILD := CGO_ENABLED=1 go build -tags "$(tags)"
GOBUILD_MINER := CGO_ENABLED=1 go build -tags "$(miner_tags)"

bin/cqld.test:
	$(GOTEST) \
		-ldflags "$(ldflags_role_bp)" \
		-o bin/cqld.test \
		github.com/CovenantSQL/CovenantSQL/cmd/cqld

bin/cqld:
	$(GOBUILD) \
		-ldflags "$(ldflags_role_bp)" \
		-o bin/cqld \
		github.com/CovenantSQL/CovenantSQL/cmd/cqld

bin/cqld.static:
	$(GOBUILD) \
		-ldflags "$(ldflags_role_bp) $(static_flags)" \
		-o bin/cqld \
		github.com/CovenantSQL/CovenantSQL/cmd/cqld

bin/cql-minerd.test:
	$(GOTEST_MINER) \
		-ldflags "$(ldflags_role_miner)" \
		-o bin/cql-minerd.test \
		github.com/CovenantSQL/CovenantSQL/cmd/cql-minerd

bin/cql-minerd:
	$(GOBUILD_MINER) \
		-ldflags "$(ldflags_role_miner)" \
		-o bin/cql-minerd \
		github.com/CovenantSQL/CovenantSQL/cmd/cql-minerd

bin/cql-minerd.static:
	$(GOBUILD_MINER) \
		-ldflags "$(ldflags_role_miner) $(static_flags)" \
		-o bin/cql-minerd \
		github.com/CovenantSQL/CovenantSQL/cmd/cql-minerd

bin/cql.test:
	$(GOTEST) \
		-ldflags "$(ldflags_role_client)" \
		-o bin/cql.test \
		github.com/CovenantSQL/CovenantSQL/cmd/cql

bin/cql:
	$(GOBUILD) \
		-ldflags "$(ldflags_role_client_simple_log)" \
		-o bin/cql \
		github.com/CovenantSQL/CovenantSQL/cmd/cql

bin/cql.static:
	$(GOBUILD) \
		-ldflags "$(ldflags_role_client_simple_log) $(static_flags)" \
		-o bin/cql \
		github.com/CovenantSQL/CovenantSQL/cmd/cql

bin/cql-fuse:
	$(GOBUILD) \
		-ldflags "$(ldflags_role_client_simple_log)" \
		-o bin/cql-fuse \
		github.com/CovenantSQL/CovenantSQL/cmd/cql-fuse

bin/cql-fuse.static:
	$(GOBUILD) \
		-ldflags "$(ldflags_role_client_simple_log) $(static_flags)" \
		-o bin/cql-fuse \
		github.com/CovenantSQL/CovenantSQL/cmd/cql-fuse

bin/cql-mysql-adapter:
	$(GOBUILD) \
		-ldflags "$(ldflags_role_client)" \
		-o bin/cql-mysql-adapter \
		github.com/CovenantSQL/CovenantSQL/cmd/cql-mysql-adapter

bin/cql-mysql-adapter.static:
	$(GOBUILD) \
		-ldflags "$(ldflags_role_client) $(static_flags)" \
		-o bin/cql-mysql-adapter \
		github.com/CovenantSQL/CovenantSQL/cmd/cql-mysql-adapter

bin/cql-proxy:
	$(GOBUILD) \
		-ldflags "$(ldflags_role_client)" \
		-o bin/cql-proxy \
		github.com/CovenantSQL/CovenantSQL/cmd/cql-proxy

bin/cql-proxy.static:
	$(GOBUILD) \
		-ldflags "$(ldflags_role_client) $(static_flags)" \
		-o bin/cql-proxy \
		github.com/CovenantSQL/CovenantSQL/cmd/cql-proxy

bp: bin/cqld.test bin/cqld

miner: bin/cql-minerd.test bin/cql-minerd

client: bin/cql bin/cql.test bin/cql-fuse bin/cql-mysql-adapter bin/cql-proxy

all: bp miner client

build-release: bin/cqld bin/cql-minerd bin/cql bin/cql-fuse bin/cql-mysql-adapter bin/cql-proxy

# This should only called in alpine docker builder
build-release-static: bin/cqld.static bin/cql-minerd.static bin/cql.static \
	bin/cql-fuse.static bin/cql-mysql-adapter.static bin/cql-proxy.static

release:
ifeq ($(unamestr),Linux)
	if [ -f /.dockerenv ]; then \
		make -j$(JOBS) build-release-static; \
	else \
		make alpine_release; \
	fi
else
	make -j$(JOBS) build-release
	tar czvf app-bin.tgz bin/cqld bin/cql-minerd bin/cql bin/cql-fuse bin/cql-mysql-adapter bin/cql-proxy
endif

android-release: status
	docker build \
		--tag $(BUILDER):android-$(VERSION) \
		-f docker/android-builder.Dockerfile \
		.
	temp_container=$$(docker create $(BUILDER):android-$(VERSION)) ; \
	docker cp $${temp_container}:/CovenantSQL.tar.gz CovenantSQL-android-$(VERSION).tar.gz && \
	docker rm $${temp_container} && \
	docker rmi $(BUILDER):android-$(VERSION)

clean:
	rm -rf bin/cql*
	rm -f *.cover.out
	rm -f coverage.txt

.PHONY: status start stop logs push push_testnet clean \
	bin/cqld.test bin/cqld bin/cql-minerd.test bin/cql-minerd \
	bin/cql bin/cql.test bin/cql-fuse bin/cql-mysql-adapter bin/cql-proxy \
	release android-release
