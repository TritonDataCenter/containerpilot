{ pkgs ? import <nixpkgs> {} }:

pkgs.mkShell {
  buildInputs = with pkgs; [
    gnumake
    go_1_17
  ];
  shellHook = ''
    echo "Ready for go development!"
  '';
}
