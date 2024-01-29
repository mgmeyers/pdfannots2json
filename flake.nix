{
  description = "pdfannots2json";

  # NixOs commands:
  #   + Run in dev shell: nix develop
  #   + Install: nix profile install ./#pdfannots2json

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixpkgs-unstable";
    flake-utils.url = github:numtide/flake-utils;
  };

  outputs = { self, nixpkgs, flake-utils, ... }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        pdfannots2json = pkgs.buildGoModule {
          pname = "pdfannots2json";
          version = "1.0.15";  # Replace with the actual version

          # Pointing src to the local directory of your project
          src = ./.;  # Replace with the actual path to your local fork

          # Since we are using a local source, vendorSha256 may not be necessary.
          # If buildGoModule requires it, set it to a dummy value and adjust according to the error message.
          vendorHash = null;
          
          # Customize build steps if necessary
          buildPhase = ''
            runHook preBuild
            go build -o $out/bin/pdfannots2json
            runHook postBuild
          '';
          
          meta = with pkgs.lib; {
            homepage = "https://github.com/mgmeyers/pdfannots2json";
            license = licenses.mit;
            description = "A tool to convert PDF annotations to JSON";
          };
        };
      in
      {
        packages.pdfannots2json = pdfannots2json;
        defaultPackage = pdfannots2json;

        # development environment
        devShells.default = pkgs.mkShell {
          nativeBuildInputs = with pkgs; [
            # Go
            go
          ];

          buildInputs = with pkgs; [
            pdfannots2json
          ];
        };
      }
    );
}
