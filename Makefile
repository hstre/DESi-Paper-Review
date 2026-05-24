DESI_PORT ?= 8765
WEB_ADDR  ?= :8080
DESI_URL  := http://127.0.0.1:$(DESI_PORT)
GO_ENV    := GOPROXY=off GOFLAGS=-mod=mod

.PHONY: help install build test test-py test-go service web dev check clean

help:
	@echo "DESi Paper Review - make targets:"
	@echo "  install   install desi-governance (if ../DESi present) + this package + build web"
	@echo "  build     build the Go web binary (web/dpr-web)"
	@echo "  test      run Python and Go tests"
	@echo "  service   run the Python DESi governance microservice (foreground)"
	@echo "  web       build and run the Go web UI (needs the service running)"
	@echo "  dev       run service + web UI together (Ctrl-C stops both)"
	@echo "  check     full health gate: service + python doctor + go self-check"
	@echo "  clean     remove build artifacts"

install:
	@if [ -d ../DESi ]; then pip install -e ../DESi; else echo "note: hstre/DESi not found at ../DESi; install desi-governance manually"; fi
	pip install -e ".[test]"
	cd web && $(GO_ENV) go build ./...

build:
	cd web && $(GO_ENV) go build -o dpr-web .

test: test-py test-go

test-py:
	python3 -m pytest -q

test-go:
	cd web && $(GO_ENV) go test ./...

service:
	python3 service/desi_service.py --port $(DESI_PORT)

web: build
	./web/dpr-web -addr $(WEB_ADDR) -desi $(DESI_URL)

dev:
	DESI_PORT=$(DESI_PORT) WEB_ADDR=$(WEB_ADDR) ./scripts/run_dev.sh

check:
	DESI_PORT=$(DESI_PORT) ./scripts/check.sh

clean:
	rm -f web/dpr-web
	cd web && $(GO_ENV) go clean
