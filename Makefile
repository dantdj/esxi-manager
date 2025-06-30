.PHONY: help setup build-frontend build-backend build dev-frontend dev-backend dev clean install-deps

# Default target
help:
	@echo "Available targets:"
	@echo "  setup          - Initial project setup"
	@echo "  install-deps   - Install frontend dependencies"
	@echo "  build-frontend - Build React app for production"
	@echo "  build-backend  - Build Go binary"
	@echo "  build          - Build both frontend and backend"
	@echo "  dev-frontend   - Start frontend dev server (port 5173)"
	@echo "  dev-backend    - Start backend dev server (port 8080)"
	@echo "  dev            - Start both dev servers concurrently"
	@echo "  clean          - Clean build artifacts"

# Initial setup
setup: install-deps
	@echo "Project setup complete!"

# Install frontend dependencies
install-deps:
	@echo "Installing frontend dependencies..."
	cd web && npm install

# Build frontend for production
build-frontend:
	@echo "Building frontend..."
	cd web && npm run build
	@echo "Frontend build complete!"

# Build Go backend
build-backend:
	@echo "Building backend..."
	go build -o esxi-manager main.go
	@echo "Backend build complete!"

# Build everything
build: build-frontend build-backend
	@echo "Full build complete!"

# Start frontend dev server
dev-frontend:
	@echo "Starting frontend dev server on http://localhost:5173"
	cd web && npm run dev

# Start backend dev server
dev-backend:
	@echo "Starting backend dev server on http://localhost:8080"
	go run main.go

# Start both dev servers (requires 'concurrently' to be installed)
dev:
	@echo "Starting both dev servers..."
	@if command -v concurrently >/dev/null 2>&1; then \
		concurrently -n "backend,frontend" -c "blue,green" \
			"make dev-backend" \
			"make dev-frontend"; \
	else \
		echo "Installing concurrently globally..."; \
		npm install -g concurrently; \
		concurrently -n "backend,frontend" -c "blue,green" \
			"make dev-backend" \
			"make dev-frontend"; \
	fi

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf web/dist/
	rm -f esxi-manager
	@echo "Clean complete!"
