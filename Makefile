.DEFAULT_GOAL := build

.PHONY: build integration-tests e2e-tests

BUILD_DIR = build/

build:

	GOOS=linux GOARCH=amd64 go build -o ${BUILD_DIR}lambda ./cmd/lambda
	@cd ${BUILD_DIR} \
		&& if [ -f lambda.zip ]; then rm lambda.zip; fi \
		&& zip lambda.zip -r . \
		&& cd -

integration-tests:
ifndef AWS_REGION
	@echo variable AWS_REGION unset
	false
endif
ifndef AWS_PROFILE
	@echo variable AWS_PROFILE unset
	false
endif
ifndef BUCKET_NAME
	@echo variable BUCKET_NAME unset
	false
endif
ifndef VERIFIED_DOMAIN
	@echo variable VERIFIED_DOMAIN unset
	false
endif
	go test -tags="integration" ./...
