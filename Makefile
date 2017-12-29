COMMANDS?=ping
# Use V=<num> to specify verbosity
VERBOSE_1 := -v
VERBOSE_2 := -v -x

.PHONY: clean clean-deps deepclean build

build:
	for target in $(COMMANDS); do \
		$(BUILD_ENV_FLAGS) go build $(VERBOSE_$(V)) -o bin/$$target ./cmd/$$target; \
	done

clean:
	rm -rf ./bin
