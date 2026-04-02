.PHONY: measure

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
