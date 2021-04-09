
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get

GIT_TAG=$(shell git tag -l | head -n1)
GIT_COMMIT_HASH:=$(shell git rev-parse HEAD)
DATE:=$(shell date)
BENCHTEST="BenchmarkKiterunnerEngineRunGolandSmall100$$"
PACKAGE="./benchmark/"
COUNT=5
SHORTBUILDFLAGS=-extld 'g++' -extldflags '-static'
BUILDFLAGS="$(SHORTBUILDFLAGS) -X 'github.com/assetnote/kiterunner/cmd/kiterunner/cmd.Version=$(GIT_TAG)' -X 'github.com/assetnote/kiterunner/cmd/kiterunner/cmd.Commit=$(GIT_COMMIT_HASH)' -X 'github.com/assetnote/kiterunner/cmd/kiterunner/cmd.Date=$(DATE)'"

build:
	mkdir -p dist
	$(GOBUILD) -ldflags $(BUILDFLAGS) -o dist/kr ./cmd/kiterunner

build-linux:
	mkdir -p dist
	GOOS=linux $(GOBUILD) -tags "netgo osusergo" -a -ldflags $(BUILDFLAGS) -o dist/kr-linux ./cmd/kiterunner

build-bench-linux:
	mkdir -p dist
	GOOS=linux $(GOBUILD) -tags "netgo osusergo" -a -ldflags "$(SHORTBUILDFLAGS)" -o dist/benchserv ./cmd/testServer/main.go

bench-mem:
	ulimit -n 20000 && \
		$(GOTEST) $(PACKAGE) -tags=integration -bench=$(BENCHTEST) -count=$(COUNT) -benchmem -run='^$$' -memprofile=/tmp/$(BENCHTEST).mem.pprof | tee ./testing/$(BENCHTEST).mem-benchstat.txt

bench-up:
	ulimit -n 20000 && \
		$(GOCMD) run ./cmd/testServer/main.go -p=14000-15001

increase-ulimit:
	sudo launchctl limit maxfiles 1048576 2048576
	ulimit -n 20000

profile:
	bash hack/profile.sh


gen-proto:
	bash ./hack/gen-proto.sh



