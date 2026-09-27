{
  description = "codemd devShell and package";
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs?rev=a3116115851d68b8952a2a4221cc25a84e56b532";
    systems.url = "github:nix-systems/default";
    flake-utils = {
      url = "github:numtide/flake-utils";
      inputs.systems.follows = "systems";
    };
  };
  outputs = {
    nixpkgs,
    flake-utils,
    ...
  }:
    flake-utils.lib.eachDefaultSystem (
      system: let
        pkgs = nixpkgs.legacyPackages.${system};
        codemd = pkgs.buildGoModule {
          pname = "codemd";
          version = "0.1.0";
          src = ./.;
          subPackages = ["cmd/codemd"];
          vendorHash = "sha256-g+yaVIx4jxpAQ/+WrGKxhVeliYx7nLQe/zsGpxV4Fn4=";
        };
      in {
        packages.default = codemd;
        packages.codemd = codemd;
        apps.default = {
          type = "app";
          program = "${codemd}/bin/codemd";
          meta.description = "Resolve code references embedded in Markdown";
        };
        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            go
            gopls
          ];
        };
      }
    );
}
