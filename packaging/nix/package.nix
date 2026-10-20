{ lib, buildGoModule, fetchFromGitHub }:

buildGoModule rec {
  pname = "rbxcb-tui";
  version = "0.1.0";

  src = fetchFromGitHub {
    owner = "Ryuzaki5100";
    repo = "rbxcb-tui";
    rev = "v${version}";
    hash = lib.fakeHash; # replace with the real source hash
  };

  vendorHash = lib.fakeHash; # replace with the real vendor hash

  subPackages = [ "cmd/rbxcb-tui" ];
  ldflags = [ "-s" "-w" "-X" "main.version=${version}" ];
  CGO_ENABLED = 0;

  meta = with lib; {
    description = "A terminal UI for solving 3x3x3 Rubik's cubes";
    homepage = "https://github.com/Ryuzaki5100/rbxcb-tui";
    license = licenses.mit;
    mainProgram = "rbxcb-tui";
    platforms = platforms.unix;
  };
}
