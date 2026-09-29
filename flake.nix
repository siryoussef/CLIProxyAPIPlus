{
  description = "Go 1.26+ proxy server providing OpenAI/Gemini/Claude/Codex compatible APIs";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = {
    self,
    nixpkgs,
    flake-utils,
  }:
    flake-utils.lib.eachDefaultSystem (
      system: let
        pkgs = import nixpkgs {inherit system;};

        cliproxyapi = pkgs.buildGoModule {
          pname = "cliproxyapi";
          version = "unstable";
          src = ./.;

          vendorHash = "sha256-P+0dbN+eKoOSBpJfo4hq86ZbWKUel+5biCBGgePm88k=";

          subPackages = ["cmd/server"];

          postInstall = ''
            mv $out/bin/server $out/bin/cliproxyapi
          '';

          meta = with pkgs.lib; {
            description = "Go proxy server providing OpenAI/Gemini/Claude/Codex compatible APIs";
            homepage = "https://github.com/router-for-me/CLIProxyAPI";
            license = licenses.mit;
            maintainers = [];
          };
        };
      in {
        packages.default = cliproxyapi;
        packages.cliproxyapi = cliproxyapi;

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            gopls
            gotools
          ];
        };
      }
    )
    // {
      overlays.default = final: prev: {
        cliproxyapi = self.packages.${prev.system}.default;
      };

      nixosModules.default = {
        config,
        lib,
        pkgs,
        ...
      }: let
        cfg = config.services.cliproxyapi;
      in {
        options.services.cliproxyapi = {
          enable = lib.mkEnableOption "CLI Proxy API service";
          package = lib.mkOption {
            type = lib.types.package;
            default = self.packages.${pkgs.system}.default;
            description = "The cliproxyapi package to use.";
          };
          configFile = lib.mkOption {
            type = lib.types.nullOr lib.types.path;
            default = null;
            description = "Path to the config.yaml file.";
          };
          port = lib.mkOption {
            type = lib.types.port;
            default = 8080;
            description = "Port to listen on.";
          };
        };

        config = lib.mkIf cfg.enable {
          systemd.services.cliproxyapi = {
            description = "CLI Proxy API Service";
            wantedBy = ["multi-user.target"];
            after = ["network.target"];
            environment = {
              PROXY_PORT = toString cfg.port;
            };
            serviceConfig = {
              ExecStart =
                ''${cfg.package}/bin/cliproxyapi''
                + lib.optionalString (cfg.configFile != null) " --config ${cfg.configFile}";
              Restart = "always";
              DynamicUser = true;
            };
          };
        };
      };

      homeManagerModules.default = {
        config,
        lib,
        pkgs,
        ...
      }: let
        cfg = config.services.cliproxyapi;
      in {
        options.services.cliproxyapi = {
          enable = lib.mkEnableOption "CLI Proxy API service";
          package = lib.mkOption {
            type = lib.types.package;
            default = self.packages.${pkgs.system}.default;
            description = "The cliproxyapi package to use.";
          };
          configFile = lib.mkOption {
            type = lib.types.nullOr lib.types.path;
            default = null;
            description = "Path to the config.yaml file.";
          };
          port = lib.mkOption {
            type = lib.types.port;
            default = 8080;
            description = "Port to listen on.";
          };
        };

        config = lib.mkIf cfg.enable {
          systemd.user.services.cliproxyapi = {
            Unit = {
              Description = "CLI Proxy API Service";
              After = ["network.target"];
            };
            Install = {
              WantedBy = ["default.target"];
            };
            Service = {
              Environment = "PROXY_PORT=${toString cfg.port}";
              ExecStart =
                ''${cfg.package}/bin/cliproxyapi''
                + lib.optionalString (cfg.configFile != null) " --config ${cfg.configFile}";
              Restart = "always";
            };
          };
        };
      };
    };
}
