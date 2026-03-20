include Makefile.variables
include Makefile.precommit
include Makefile.docker
include example.env

SERVICE = bborbe/go-skeleton

run:
	@go run -mod=mod main.go \
	-sentry-dsn="$(shell teamvault-url --teamvault-config ~/.teamvault.json --teamvault-key=${SENTRY_DSN_KEY})" \
	-listen="localhost:${SKELETON_PORT}" \
	-kafka-brokers="${KAFKA_BROKERS}" \
	-datadir="data" \
	-batch-size="100" \
	-v=2

deps:
	go install github.com/bborbe/teamvault-utils/cmd/teamvault-config-parser@latest
	go install github.com/bborbe/teamvault-utils/cmd/teamvault-file@latest
	go install github.com/bborbe/teamvault-utils/cmd/teamvault-url@latest
	go install github.com/bborbe/teamvault-utils/cmd/teamvault-username@latest
	go install github.com/bborbe/teamvault-utils/cmd/teamvault-password@latest
	go install github.com/onsi/ginkgo/v2/ginkgo@v2.25.3
	sudo port install trivy

formatenv:
	cat example.env | sort > c
	mv c example.env
