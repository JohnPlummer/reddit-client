.PHONY: run-basic run-comprehensive run-examples test tidy tidy-examples tidy-all lint lint-examples lint-all check

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

# Run go fmt in root project
lint:
	@echo "Running go fmt in root project..."
	go fmt ./...

# Run go fmt in examples
lint-examples:
	@echo "Running go fmt in examples..."
	cd examples/basic && go fmt ./...
	cd examples/comprehensive && go fmt ./...

# Run go fmt everywhere
lint-all: lint lint-examples

# Run all checks: tidy, lint, test, and run examples
check:
	@echo "Running all checks..."
	@echo "Step 1: Running tidy-all..."
	@make tidy-all
	@echo "Step 2: Running lint-all..."
	@make lint-all
	@echo "Step 3: Running tests..."
	@make test
	@echo "Step 4: Running examples..."
	@make run-examples
	@echo "All checks completed successfully!"
