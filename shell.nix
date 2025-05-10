{ pkgs ? import <nixpkgs> {} }:

pkgs.mkShell {
  name = "wails-dev-env";

  packages = with pkgs; [
    gtk3
    webkitgtk
    pkg-config

    gtk3.dev
    webkitgtk.dev
  ];

  # env variables for gtk
  shellHook = ''
    export XDG_DATA_DIRS="$GSETTINGS_SCHEMAS_PATH:${pkgs.gtk3}/share/gsettings-schemas/${pkgs.gtk3.name}:$XDG_DATA_DIRS"
    export GI_TYPELIB_PATH="${pkgs.gtk3}/lib/girepository-1.0:${pkgs.webkitgtk}/lib/girepository-1.0:$GI_TYPELIB_PATH"
    export LD_LIBRARY_PATH="${pkgs.gtk3}/lib:${pkgs.webkitgtk}/lib:$LD_LIBRARY_PATH"
  '';
}
