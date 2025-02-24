.PHONY: run-basic run-comprehensive run-examples test tidy tidy-examples

# Run the basic example
run-basic:
	@echo "Running basic example..."
	cd examples/basic && go run main.go

# Run the comprehensive example
run-comprehensive:
	@echo "Running comprehensive example..."
	cd examples/comprehensive && go run main.go \
		--subreddit=golang \
		--limit=5 \
		--sort=new \
		--max-pages=1 \
		--log-level=info

# Run both examples
run-examples: run-basic run-comprehensive

# Run tests using Ginkgo
test:
	@echo "Running tests..."
	ginkgo -v ./...

# Run go mod tidy in root project
tidy:
	@echo "Running go mod tidy in root project..."
	go mod tidy

# Run go mod tidy in examples
tidy-examples:
	@echo "Running go mod tidy in examples..."
	cd examples/basic && go mod tidy
	cd examples/comprehensive && go mod tidy

# Run go mod tidy everywhere
tidy-all: tidy tidy-examples
