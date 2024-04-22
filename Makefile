.PHONY: help chores clean coverage pkgsite report test vuln

export GOTOOLCHAIN=local

help: ## list available targets
	@# Derived from Gomega's Makefile (github.com/onsi/gomega) under MIT License
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-16s\033[0m %s\n", $$1, $$2}'

clean: ## cleans up build and testing artefacts
	rm -f coverage.html coverage.out __debug_bin mobydig

coverage: ## gathers coverage and updates README badge
	@scripts/cov.sh

pkgsite: ## serves Go documentation on port 6060
	@echo "navigate to: http://localhost:6060/github.com/siemens/mobydig"
	@scripts/pkgsite.sh

report: ## runs goreportcard on this module
	@scripts/goreportcard.sh

test: ## runs unit tests
	go test -v -p=1 -count=1 -race -exec sudo ./...

vuln: ## runs govulncheck
	@scripts/vuln.sh

chores: ## updates Go binaries and NPM helper packages if necessary
	@scripts/chores.sh
