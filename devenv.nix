{ pkgs, ... }:
{
  languages.go = {
    enable = true;
    package = pkgs.go_1_27;
  };

  packages = with pkgs; [
    golangci-lint
    gotools
    gopls
    just
  ];

  env.CGO_ENABLED = "0";

  scripts = {
    rbxcb-tui.exec = "go run ./cmd/rbxcb-tui \"$@\"";
    tui.exec = "go run ./cmd/rbxcb-tui";
    demo.exec = "go run ./cmd/rbxcb-tui --demo";
    test.exec = "go vet ./... && go test ./...";
    lint.exec = "golangci-lint run";
  };

  enterTest = ''
    go vet ./...
    go test ./...
  '';
}
