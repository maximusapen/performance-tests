GO111MODULE := on
GOPRIVATE := github.ibm.com
export

GOPACKAGES=$(shell go list ./... | grep -E -v 'k8s.io')
GOFILES=$(shell find . -type f -name '*.go' -not -path "./vendor/*" -not -path "./k8s.io/*")

.PHONY: all
all: deps fmt vet lint

.PHONY: setup
setup:
	@pre-commit install; \
	if [ $$? != 0 ]; then echo "Pre-commit tool is required for this repo. See https://github.ibm.com/alchemy-containers/armada-performance/blob/master/README.md" && exit 1; fi;

.PHONY: lintall
lintall: vet fmt lint

.PHONY: deps
deps:
	go get ${GOPACKAGES}
	go mod tidy -compat=1.19
	go install \
		golang.org/x/lint/golint@latest

.PHONY: fmt
fmt:
	@if [ -n "$$(gofmt -l ${GOFILES})" ]; then echo 'Please run gofmt -l -w on your code.' && exit 1; fi

.PHONY: dofmt
dofmt:
	gofmt -l -w ${GOFILES}

.PHONY: lint-copyright
lint-copyright:
	./scripts/lint_copyright.sh

.PHONY: vet
vet:
	CGO_ENABLED=0 GOOS=linux go vet ${GOPACKAGES}

.PHONY: lint
lint:
	$(GOPATH)/bin/golint -set_exit_status=true ${GOPACKAGES}

.PHONY: sec
sec:
	go get -v github.com/securego/gosec/cmd/gosec/
	gosec -exclude=G104 -quiet ./...

.PHONY: test
test:
	#echo 'mode: atomic' > cover.out && echo "./etcd/etcd-backup/" | xargs -n1 -I{} sh -c 'go test -v -race -covermode=atomic -coverprofile=coverage.tmp {} && tail -n +2 coverage.tmp >> cover.out' && rm     coverage.tmp

.PHONY: coverage
coverage:
	# no test so not coverage - go tool cover -html=cover.out -o=cover.html
	# go tool cover -html=cover.out -o=cover.html

.PHONY: buildgo
buildgo:
	cd api/cruiser_churn; CGO_ENABLED=0 GOOS=linux go build -ldflags "-s" -a -installsuffix cgo .
	cd api/cruiser_mon; CGO_ENABLED=0 GOOS=linux go build -ldflags "-s" -a -installsuffix cgo .
	cd api/kubernetes/k8s-metrics; CGO_ENABLED=0 GOOS=linux go build -ldflags "-s" -a -installsuffix cgo .
	cd armada-perf-client2; CGO_ENABLED=0 GOOS=linux go build -ldflags "-s" -a -installsuffix cgo .
	cd automation/displayTestResults; CGO_ENABLED=0 GOOS=linux go build -ldflags "-s" -a -installsuffix cgo .
	cd etcd/etcd-driver; CGO_ENABLED=0 GOOS=linux go build -ldflags "-s" -a -installsuffix cgo .
	cd etcd/lease_test; CGO_ENABLED=0 GOOS=linux go build -ldflags "-s" -a -installsuffix cgo .
	cd httpperf/imageCreate/httpperf; CGO_ENABLED=0 GOOS=linux go build -ldflags "-s" -a -installsuffix cgo .
	cd incluster-apiserver/imageCreate/incluster-apiserver; CGO_ENABLED=0 GOOS=linux go build -ldflags "-s" -a -installsuffix cgo .
	cd metrics/alerting; CGO_ENABLED=0 GOOS=linux go build -ldflags "-s" -a -installsuffix cgo .
	cd metrics/bluemix/send-to-bm; CGO_ENABLED=0 GOOS=linux go build -ldflags "-s" -a -installsuffix cgo .
	cd metrics/bluemix/send-file-to-Influx; CGO_ENABLED=0 GOOS=linux go build -ldflags "-s" -a -installsuffix cgo .
	cd metrics/bluemix/send-parallel-files-to-Influx; CGO_ENABLED=0 GOOS=linux go build -ldflags "-s" -a -installsuffix cgo .
	cd metrics/carrier/carrier-collector; CGO_ENABLED=0 GOOS=linux go build -ldflags "-s" -a -installsuffix cgo
	cd metrics/cruiser/cruiser-collector; CGO_ENABLED=0 GOOS=linux go build -ldflags "-s" -a -installsuffix cgo .
	cd metrics/kubernetes-e2e; CGO_ENABLED=0 GOOS=linux go build -ldflags "-s" -a -installsuffix cgo .
	cd metrics/kubernetes-netperf; CGO_ENABLED=0 GOOS=linux go build -ldflags "-s" -a -installsuffix cgo .
	cd metrics/prometheus; CGO_ENABLED=0 GOOS=linux go build -ldflags "-s" -a -installsuffix cgo .
	cd metrics/jmeter/sendJmeterResultsBM; CGO_ENABLED=0 GOOS=linux go build -ldflags "-s" -a -installsuffix cgo .
	cd k8s-netperf/netperf; CGO_ENABLED=0 GOOS=linux go build -ldflags "-s" -a -installsuffix cgo .
	cd k8s-netperf/nptests; CGO_ENABLED=0 GOOS=linux go build -ldflags "-s" -a -installsuffix cgo .
	cd persistent-storage/snapshotStorage; CGO_ENABLED=0 GOOS=linux go build -ldflags "-s" -a -installsuffix cgo .
	cd registry/imageCreate/registry; CGO_ENABLED=0 GOOS=linux go build -ldflags "-s" -a -installsuffix cgo .
	cd tools/annotateBOM; CGO_ENABLED=0 GOOS=linux go build -ldflags "-s" -a -installsuffix cgo .
	cd tools/detectDefaultLevels; CGO_ENABLED=0 GOOS=linux go build -ldflags "-s" -a -installsuffix cgo .
	cd tools/cancelSubnet; CGO_ENABLED=0 GOOS=linux go build -ldflags "-s" -a -installsuffix cgo .
	cd tools/crypto; CGO_ENABLED=0 GOOS=linux go build -ldflags "-s" -a -installsuffix cgo .
	cd tools/carrier-lock; CGO_ENABLED=0 GOOS=linux go build -ldflags "-s" -a -installsuffix cgo .
	cd tools/tomlToJson; CGO_ENABLED=0 GOOS=linux go build -ldflags "-s" -a -installsuffix cgo .
	cd sysbench/imageCreate/run-sysbench; CGO_ENABLED=0 GOOS=linux go build -ldflags "-s" -a -installsuffix cgo .

# Used for Jenkins builds
.PHONY: installgo
installgo:
		cd api/cruiser_churn; CGO_ENABLED=0 GOOS=linux go install -ldflags "-s -X 'main.encryptionKey=$(STAGE_GLOBAL_ARMPERF_CRYPTOKEY)'" -a -installsuffix cgo .
		cd api/cruiser_mon; CGO_ENABLED=0 GOOS=linux go install -ldflags "-s -X 'main.encryptionKey=$(STAGE_GLOBAL_ARMPERF_CRYPTOKEY)'" -a -installsuffix cgo .
		cd api/kubernetes/k8s-metrics; CGO_ENABLED=0 GOOS=linux go install -ldflags "-s" -a -installsuffix cgo .
		cd armada-perf-client2; CGO_ENABLED=0 GOOS=linux go install -ldflags "-s" -a -installsuffix cgo .
		cd automation/displayTestResults; CGO_ENABLED=0 GOOS=linux go install -ldflags "-s" -a -installsuffix cgo .
		cd etcd/etcd-driver; CGO_ENABLED=0 GOOS=linux go install -ldflags "-s" -a -installsuffix cgo .
		cd etcd/lease_test; CGO_ENABLED=0 GOOS=linux go install -ldflags "-s" -a -installsuffix cgo .
		cd httpperf/imageCreate/httpperf; CGO_ENABLED=0 GOOS=linux go install -ldflags "-s" -a -installsuffix cgo .
		cd incluster-apiserver/imageCreate/incluster-apiserver; CGO_ENABLED=0 GOOS=linux go install -ldflags "-s" -a -installsuffix cgo .
		cd metrics/alerting; CGO_ENABLED=0 GOOS=linux go install -ldflags "-s" -a -installsuffix cgo .
		cd metrics/bluemix/send-to-bm; CGO_ENABLED=0 GOOS=linux go install -ldflags "-s" -a -installsuffix cgo .
		cd metrics/bluemix/send-file-to-Influx; CGO_ENABLED=0 GOOS=linux go install -ldflags "-s" -a -installsuffix cgo .
		cd metrics/bluemix/send-parallel-files-to-Influx; CGO_ENABLED=0 GOOS=linux go install -ldflags "-s" -a -installsuffix cgo .
		cd metrics/cruiser/cruiser-collector; CGO_ENABLED=0 GOOS=linux go install -ldflags "-s" -a -installsuffix cgo .
		cd metrics/carrier/carrier-collector; CGO_ENABLED=0 GOOS=linux go install -ldflags "-s" -a -installsuffix cgo .
		cd metrics/kubernetes-e2e; CGO_ENABLED=0 GOOS=linux go install -ldflags "-s" -a -installsuffix cgo .
		cd metrics/kubernetes-netperf; CGO_ENABLED=0 GOOS=linux go install -ldflags "-s" -a -installsuffix cgo .
		cd metrics/prometheus; CGO_ENABLED=0 GOOS=linux go install -ldflags "-s" -a -installsuffix cgo .
		cd metrics/jmeter/sendJmeterResultsBM; CGO_ENABLED=0 GOOS=linux go install -ldflags "-s" -a -installsuffix cgo .
		cd network/netperf; CGO_ENABLED=0 GOOS=linux go install -ldflags "-s" -a -installsuffix cgo .
		cd k8s-netperf/netperf; CGO_ENABLED=0 GOOS=linux go install -ldflags "-s" -a -installsuffix cgo .
		cd k8s-netperf/nptests; CGO_ENABLED=0 GOOS=linux go install -ldflags "-s" -a -installsuffix cgo .
		cd persistent-storage/snapshotStorage; CGO_ENABLED=0 GOOS=linux go install -ldflags "-s" -a -installsuffix cgo .
		cd registry/imageCreate/registry; CGO_ENABLED=0 GOOS=linux go install -ldflags "-s" -a -installsuffix cgo .
		cd tools/annotateBOM; CGO_ENABLED=0 GOOS=linux go install -ldflags "-s" -a -installsuffix cgo .
		cd tools/detectDefaultLevels; CGO_ENABLED=0 GOOS=linux go install -ldflags "-s" -a -installsuffix cgo .
		cd tools/cancelSubnet; CGO_ENABLED=0 GOOS=linux go install -ldflags "-s -X 'main.encryptionKey=$(STAGE_GLOBAL_ARMPERF_CRYPTOKEY)'" -a -installsuffix cgo .
		cd tools/crypto; CGO_ENABLED=0 GOOS=linux go install -ldflags "-s" -a -installsuffix cgo .
		cd tools/carrier-lock; CGO_ENABLED=0 GOOS=linux go install -ldflags "-s" -a -installsuffix cgo .
		cd tools/tomlToJson; CGO_ENABLED=0 GOOS=linux go install -ldflags "-s" -a -installsuffix cgo .
		cd sysbench/imageCreate/run-sysbench; CGO_ENABLED=0 GOOS=linux go install -ldflags "-s" -a -installsuffix cgo .

.PHONY: builddocker
builddocker:
	go run github.ibm.com/alchemy-containers/go-build-tools/cmd/goproxy -docker-build -- \
		docker build -t armada/perf-build .

.PHONY: buildapc2
buildapc2:
	cd armada-perf-client2; CGO_ENABLED=0 GOOS=linux go build -ldflags "-s" -a -installsuffix cgo .

.PHONY: installapc2
installapc2:
	cd armada-perf-client2; CGO_ENABLED=0 GOOS=linux go install -ldflags "-s" -a -installsuffix cgo .
