
lint:
	golangci-lint run

lint-fix:
	golangci-lint run --fix

proto:
	for PROTO_FILE in $$(find . -name '*.proto'); do \
		echo "generating codes for $$PROTO_FILE"; \
		protoc \
			--go_out=. \
			--go_opt paths=source_relative \
			--plugin protoc-gen-go="${GOPATH}/bin/protoc-gen-go" \
			--go-grpc_out=. \
			--go-grpc_opt paths=source_relative \
			--plugin protoc-gen-go-grpc="${GOPATH}/bin/protoc-gen-go-grpc" \
			$$PROTO_FILE; \
	done

build-java-executor:
	mvn install -f executors/executor-java/pom.xml

run-java-test-executor:
	mvn test-compile exec:java -Dexec.mainClass="com.github.loadmesh.TestExecutor" -Dexec.classpathScope=test -f executors/executor-java/pom.xml

test:
	go test ./...

license:
	./license-checker/license-checker.sh
