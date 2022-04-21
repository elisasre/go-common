OPERATOR_NAME := go-common
ifeq ($(USE_JSON_OUTPUT), 1)
GOTEST_REPORT_FORMAT := -json
endif

.PHONY: clean ensure build golint test deps-check sonar-scanner

clean:
	git clean -Xdf

ensure:
	go mod tidy -compat=1.17
	go mod vendor

build:
	rm -f bin/$(OPERATOR_NAME)
	go build -mod vendor -v -o bin/$(OPERATOR_NAME) .

golint:
	pre-commit run --all-files

test:
	go test -failfast -mod vendor ./*.go -v -covermode atomic -coverprofile=gotest-coverage.out $(GOTEST_REPORT_FORMAT) > gotest-report.out && cat gotest-report.out || (cat gotest-report.out; exit 1)
	git diff --exit-code go.mod go.sum

deps-check:
	ret=0; \
	dependency-check --cveValidForHours 24 --connectiontimeout 30000 --format ALL \
		--scan go.mod \
		--exclude **/vendor/** \
		--enableExperimental --failOnCVSS 11 \
		--project $(OPERATOR_NAME); ret=$$?; \
	sed "/^$$/d" dependency-check-report.csv; \
	exit $$ret

sonar-scanner:
	@sonar-scanner \
		-Dsonar.login=$(SONAR_LOGIN) \
		-Dsonar.host.url=https://rotta.saunalahti.fi \
		-Dproject.settings=.sonarprops \
		$(SONAR_OPTS)
