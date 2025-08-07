{
  description = "efm langserver env";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixpkgs-unstable";
  };

  outputs =
    { self, nixpkgs, ... }:
    let
      nixpkgsFor =
        system:
        (import nixpkgs {
          inherit system;
          config.allowUnfree = true;
        });
      forAllSystems =
        function:
        nixpkgs.lib.genAttrs [
          "x86_64-linux"
          "aarch64-linux"
          "x86_64-darwin"
          "aarch64-darwin"
        ] (system: function (nixpkgsFor system));
    in
    {
      devShells = forAllSystems (pkgs: {
        default = pkgs.mkShell {
          name = "efm";
          packages = with pkgs; [
            nodePackages.prettier
          ];
          inputsFrom = [ self.packages.${pkgs.system}.default ];
        };
      });
      formatter = forAllSystems (pkgs: pkgs.nixpkgs-fmt);

      packages = forAllSystems (
        pkgs:
        let
          pname = "efm-langserver";
          postInstallScript =
            os:
            let
              binary = if os == "windows" then "${pname}.exe" else pname;
            in
            ''
              binary=$(find $out -type f -name ${binary})
              mv $binary $out/${binary}
              rm -rf $out/bin/*
              mv $out/${binary} $out/bin/${binary}
              sha256sum $out/bin/${binary} > $out/bin/${binary}.sha256
            '';
        in
        rec {
          default = pkgs.buildGoModule {
            inherit pname;
            version = "0.0.1";

            env.CGO_ENABLED = 0;
            src = pkgs.lib.cleanSource ./.;

            nativeBuildInputs = [ pkgs.golangci-lint ];
            vendorHash = null;
            preBuild = ''
              export GOLANGCI_LINT_CACHE="$(mktemp -d)"
              golangci-lint run ./...
            '';
          };
          amd64-linux = default.overrideAttrs rec {
            env.GOOS = "linux";
            env.GOARCH = "amd64";
            postInstall = postInstallScript env.GOOS;
          };
          arm64-linux = default.overrideAttrs rec {
            env.GOOS = "linux";
            env.GOARCH = "arm64";
            postInstall = postInstallScript env.GOOS;
          };
          amd64-darwin = default.overrideAttrs rec {
            env.GOOS = "darwin";
            env.GOARCH = "amd64";
            postInstall = postInstallScript env.GOOS;
          };
          arm64-darwin = default.overrideAttrs rec {
            env.GOOS = "darwin";
            env.GOARCH = "arm64";
            postInstall = postInstallScript env.GOOS;
          };
          amd64-windows = default.overrideAttrs rec {
            env.GOOS = "windows";
            env.GOARCH = "amd64";
            postInstall = postInstallScript env.GOOS;
          };
          arm64-windows = default.overrideAttrs rec {
            env.GOOS = "windows";
            env.GOARCH = "arm64";
            postInstall = postInstallScript env.GOOS;
          };
        }
      );
    };
}
