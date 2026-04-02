.PHONY: setup uninstall measure kwa-build kwa-run

GO_CACHE_DIR := $(CURDIR)/.gocache_local

setup:
	./scripts/setup.sh

uninstall:
	./scripts/uninstall.sh

# Main benchmark target delegates to scripts/measure.sh.
measure:
	@args=""; \
	if [ -n "$(profile)" ]; then args="$$args profile=$(profile)"; fi; \
	if [ -n "$(iterations)" ]; then args="$$args iterations=$(iterations)"; fi; \
	if [ -n "$(lang)" ]; then args="$$args lang=$(lang)"; fi; \
	if [ -n "$(bench)" ]; then args="$$args bench=$(bench)"; fi; \
	if [ -n "$(gmt_dir)" ]; then args="$$args gmt_dir=$(gmt_dir)"; fi; \
	if [ -n "$(uri)" ]; then args="$$args uri=$(uri)"; fi; \
	./scripts/measure.sh $$args

kwa-build:
	@mkdir -p "$(GO_CACHE_DIR)" kwa/build
	cd kwa && GOCACHE="$(GO_CACHE_DIR)" go build -o build/kwa ./cmd/main.go

kwa-run:
	@mkdir -p "$(GO_CACHE_DIR)"
	cd kwa && GOCACHE="$(GO_CACHE_DIR)" go run ./cmd/main.go
