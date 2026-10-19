{
  description = "rbxcb-tui: a Bubble Tea TUI that solves 3x3x3 Rubik's cubes";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs =
    { self, nixpkgs }:
    let
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];
      forAll = f: nixpkgs.lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});
    in
    {
      packages = forAll (pkgs: {
        default = (pkgs.buildGoModule.override { go = pkgs.go_1_27; }) {
          pname = "rbxcb-tui";
          version = self.shortRev or self.dirtyShortRev or "dev";
          src = ./.;
          vendorHash = null;
          subPackages = [ "cmd/rbxcb-tui" ];
          ldflags = [
            "-s"
            "-w"
          ];
          meta.mainProgram = "rbxcb-tui";
        };
      });

      apps = forAll (pkgs: {
        default = {
          type = "app";
          program = pkgs.lib.getExe self.packages.${pkgs.stdenv.hostPlatform.system}.default;
        };
      });

      devShells = forAll (pkgs: {
        default = pkgs.mkShell {
          packages = [
            pkgs.go_1_27
            pkgs.gopls
            pkgs.golangci-lint
            pkgs.gotools
            pkgs.just
          ];
          CGO_ENABLED = "0";
        };
      });
    };
}
