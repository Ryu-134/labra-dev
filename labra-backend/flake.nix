{
	description = "The backend flake of Labra";

	inputs = { 
		nixpkgs.url = "github:nixos/nixpkgs";
		flake-utils.url = "github:numtide/flake-utils";
	};

	outputs = { self, nixpkgs, flake-utils }:
		flake-utils.lib.eachDefaultSystem (
		system:
			let pkgs = nixpkgs.legacyPackages.${system};
			in {
				devShell = pkgs.mkShell { 
					buildInputs = [ 
							pkgs.go 
							pkgs.air
							pkgs.sqlite
					 ]; 

					shellHook = ''
						if [ ! -f .env ]; then
							echo "dotenv not found, generating one from example"
							cp .env.example .env
						fi
						
						echo "Entered flake"

					'';

				};
			});
}
