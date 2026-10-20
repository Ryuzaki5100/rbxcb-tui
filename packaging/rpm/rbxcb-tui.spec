Name:           rbxcb-tui
Version:        __VERSION__
Release:        1%{?dist}
Summary:        A terminal UI for solving 3x3x3 Rubik's cubes
License:        MIT
URL:            https://github.com/Ryuzaki5100/rbxcb-tui
Source0:        %{url}/archive/refs/tags/v%{version}.tar.gz
BuildRequires:  golang
BuildRequires:  git

%description
A keyboard- and mouse-driven terminal application for entering and solving
3x3x3 Rubik's cubes with staged solvers.

%prep
%autosetup -n rbxcb-tui-%{version}

%build
export CGO_ENABLED=0
go build -mod=vendor -trimpath \
  -ldflags "-s -w -X main.version=%{version}" \
  -o rbxcb-tui ./cmd/rbxcb-tui

%install
install -Dm755 rbxcb-tui %{buildroot}%{_bindir}/rbxcb-tui
install -Dm644 LICENSE %{buildroot}%{_licensedir}/rbxcb-tui/LICENSE
install -Dm644 README.md %{buildroot}%{_docdir}/rbxcb-tui/README.md

%files
%license LICENSE
%doc README.md
%{_bindir}/rbxcb-tui

%changelog
* Tue Oct 20 2026 Ryuzaki5100 <Ryuzaki5100@users.noreply.github.com> - __VERSION__-1
- Initial package
