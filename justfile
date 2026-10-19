default: run

run *args:
    go run ./cmd/rbxcb-tui {{args}}

demo:
    go run ./cmd/rbxcb-tui --demo

test:
    go vet ./...
    go test ./...

test-e2e:
    go test ./e2e -run E2E -v

test-e2e-long:
    E2E_SCRAMBLES=100 go test ./e2e -run E2E -v -timeout 30m

lint:
    golangci-lint run

fmt:
    gofmt -w .

build:
    go build -trimpath -ldflags "-s -w" -o bin/rbxcb-tui ./cmd/rbxcb-tui

nix-build:
    nix build
