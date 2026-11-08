{
  description = "io-surface — the I/O surface manifest gate for Go trees";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

    flake-parts = {
      url = "github:hercules-ci/flake-parts";
      inputs.nixpkgs-lib.follows = "nixpkgs";
    };

    go-nix-helpers = {
      url = "github:LarsArtmann/go-nix-helpers";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs =
    inputs@{
      flake-parts,
      ...
    }:
    let
      lib = inputs.nixpkgs.lib;

      # Everything git tracks: the gate is hermetic (stdlib-only extract),
      # but its own tests need the fixture trees (testdata/extract,
      # testdata/detectors), the golden files, and the go:embed'd schema.cue.
      packageSrc = lib.fileset.toSource {
        root = ./.;
        fileset = lib.fileset.gitTracked ./.;
      };
    in
    flake-parts.lib.mkFlake { inherit inputs; } {
      imports = [ inputs.go-nix-helpers.flakeModules.go-standard ];

      go-standard = {
        pname = "io-surface";
        description = "Extract every I/O call site from a Go tree, compare it against a CUE manifest, and fail the build on undocumented I/O — plus a manifestless io-bug-radar detector face";
        goPkgAttr = "go_1_27";
        # Linux only, matching the fleet's go-standard restriction: the
        # proxy-vendored go-modules FOD hashes differently on darwin.
        systems = [
          "x86_64-linux"
          "aarch64-linux"
        ];
        src = packageSrc;
        subPackages = [ "cmd/io-surface" ];
        vendorHash = "sha256-vPMkJMzE3/mR+eC7fxmvt4VsRHfcN2kjY9SHtXBBAHc=";
        # Match the fleet build: vendoring via the module proxy.
        proxyVendor = true;
        enableTestCheck = true;
        enableGofumpt = true;
        enableGoimports = true;
        enableNixfmt = true;
        extraMeta = {
          mainProgram = "io-surface";
        };
      };
    };
}
