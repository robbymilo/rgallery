HUGO := $(shell cd website && hvm status --printExecPathCached || hvm status --printExecPath)
TARGETARCH = $(shell go env GOARCH)
TARGETOS = $(shell go env GOOS)
FLAGS = -ldflags="-X 'main.Commit=$(shell git rev-parse HEAD)' -X 'main.Tag=$(shell git describe --exact-match --tags)'"
SHA = $(shell git rev-parse HEAD)
TZ = Europe/Stockholm
-include $(PWD)/.env

clean:
	rm -rf bin
	rm -rf testdata/data
	rm -rf testdata/cache

# frontend
.PHONY: npm
npm:
	npm ci

.PHONY: sha
sha:
	@echo '{"sha": "$(SHA)", "tag": "$(TAG)"}' > ui/version.json

.PHONY: assets
assets: sha
	cd ui && npm run build

.PHONY: prettier
prettier:
	npm run prettier

.PHONY: prettier-test
prettier-test:
	npm run prettier-test

build: build-rgallery build-rgallery-resize build-rgallery-geo

build-rgallery:
	CGO_ENABLED=0 GOOS=$(TARGETOS) GOARCH=$(TARGETARCH) go build $(FLAGS) -o bin/rgallery_$(VERSION)$(TARGETOS)-$(TARGETARCH) github.com/robbymilo/rgallery/cmd/rgallery

build-rgallery-resize:
	CGO_ENABLED=0 GOOS=$(TARGETOS) GOARCH=$(TARGETARCH) go build $(FLAGS) -o bin/rgallery-resize_$(VERSION)$(TARGETOS)-$(TARGETARCH) github.com/robbymilo/rgallery/cmd/rgallery-resize

build-rgallery-geo:
	CGO_ENABLED=0 GOOS=$(TARGETOS) GOARCH=$(TARGETARCH) go build $(FLAGS) -o bin/rgallery-geo_$(VERSION)$(TARGETOS)-$(TARGETARCH) github.com/robbymilo/rgallery/cmd/rgallery-geo

docker-run:
	docker compose up --build

docker-run-scalable:
	docker compose -f docker-compose-scalable.yml up --scale rgallery-resize=3 --build

DEV_FLAGS=--location-dataset=Countries10

# local dev
.PHONY: run
run:
	TZ=$(TZ) go run $(FLAGS) ./cmd/rgallery/main.go -dev $(DEV_FLAGS)

resize:
	go run ./cmd/rgallery-resize/main.go

geo:
	go run ./cmd/rgallery-geo/main.go

run-all:
	make -j 3 run resize geo RGALLERY_RESIZE_SERVICE=http://localhost:3001 RGALLERY_LOCATION_SERVICE=http://localhost:3002 TZ=$(TZ)

# run make test before to get mod time correct
run-test: clean
	go run $(FLAGS) ./cmd/rgallery/main.go --disable-auth --media testdata/media --cache testdata/cache --data testdata/data --config testdata/config/config.yml --memories=false  TZ=$(TZ)

.PHONY: test
test: clean
	go test ./... -v

lint:
	golangci-lint run --timeout=10m

scan:
	go run $(FLAGS) ./cmd/rgallery/main.go --location_dataset Countries10 scan

.PHONY: install-mac
install-mac: build-darwin
	sudo mv ./bin/rgallery_darwin-universal /usr/local/bin/rgallery \
	&& sudo chmod +x /usr/local/bin/rgallery

website-build-sha:
	printf "/*\n  build: $(SHA)" > website/content/_headers

website: website-build-sha
	cd website && npm ci
	cd website && hugo --gc --minify

server:
	cd website && (hvm use --useVersionInDotFile || true) && $(HUGO) server

typecheck:
	npm run typecheck

vite: sha
	cd ui && npm run dev

preview:
	cd ui && npm run build && npm run preview
