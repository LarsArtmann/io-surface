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
        vendorHash = "sha256-HPggmhV7yPt441HeeIf0WMinA/r2+YrID9tydlBzcsI=";
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

      # Fixture trees (testdata/extract, testdata/detectors) are contract
      # data: parseable-only Go with line-pinned test expectations. goimports
      # prunes the fixture's unresolvable seam import and rewrites the embed
      # import, breaking seam tests and location pins — formatters must not
      # touch testdata. The CI gofmt gate stays whole-tree. treefmt v2.6
      # global excludes match root-relative globs, so the root-level
      # testdata/** shape is the working form (leading **/ does not match);
      # nested testdata dirs hold only .golden/.md files, which no formatter
      # includes.
      perSystem.treefmt.settings.excludes = [ "testdata/**" ];
    };
}
