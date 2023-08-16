# Download wanted go.mk version automatically if not present.
BASE_VERSION  := 2a6770c
BASE_MAKE     := go-${BASE_VERSION}.mk
FETCH_BASE_MAKE	= $(shell gh api -H 'Accept: application/vnd.github.v3.raw' 'repos/elisasre/baseconfig/contents/go.mk?ref=${BASE_VERSION}' > ${BASE_MAKE})
ifeq ($(wildcard ${BASE_MAKE}),)
TRIGGER := ${FETCH_BASE_MAKE}
endif

include ${BASE_MAKE}

.PHONY: clean update run cover-info

validate-go-mk:
	@echo Updating go.mk: ${FETCH_BASE_MAKE}
	git diff --exit-code -- ${BASE_MAKE}

clean:
	git clean -Xdf

update:
	go get -u

cover-info: go-merge-coverages
	go tool cover -func ${GO_TOTAL_COV_FILE}
