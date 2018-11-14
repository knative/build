#This makefile is used by ci-operator

CGO_ENABLED=0
GOOS=linux

install:
	go install ./cmd/controller/ ./cmd/creds-init ./cmd/git-init/ ./cmd/nop ./cmd/webhook
.PHONY: install

test-install:
	go install ./test/panic/
.PHONY: test-install

test-e2e:
	sh openshift/e2e-tests-openshift.sh
.PHONY: test-e2e
