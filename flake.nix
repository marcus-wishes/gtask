{
  description = "gtask - Minimal CLI for Google Tasks";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        packages = {
          gtask = pkgs.buildGoModule {
            pname = "gtask";
            version = "0.1.0";
            src = ./.;
            
            vendorHash = pkgs.lib.fakeHash; # Set to null if using vendor/, otherwise use lib.fakeHash first
            #vendorHash = "sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="; # real hash
            
            subPackages = [ "cmd/gtask" ];

            meta = with pkgs.lib; {
              description = "Minimal CLI for Google Tasks";
              homepage = "https://github.com/markus-wishes/gtask";
              license = licenses.mit; # adjust as needed
              mainProgram = "gtask";
            };
          };
          
          default = self.packages.${system}.gtask;
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            gopls
            gotools
          ];
        };
      }
    );
}