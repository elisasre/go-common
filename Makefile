OPERATOR_NAME := go-common

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

