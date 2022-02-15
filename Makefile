OPERATOR_NAME := go-common

ensure:
	GO111MODULE=on go mod tidy -compat=1.17
	GO111MODULE=on go mod vendor

build:
	rm -f bin/$(OPERATOR_NAME)
	GO111MODULE=on go build -mod vendor -v -o bin/$(OPERATOR_NAME) .

golint:
	pre-commit run --all-files

test:
	GO111MODULE=on go test -failfast -mod vendor ./*.go -v -covermode atomic -coverprofile=gotest-coverage.out $(GOTEST_REPORT_FORMAT) > gotest-report.out && cat gotest-report.out || (cat gotest-report.out; exit 1)
	git diff --exit-code go.mod go.sum

