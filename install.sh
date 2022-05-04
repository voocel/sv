#!/bin/sh

set -eu

GOROOT=${GOROOT:-$HOME/.sv/go}
if [ "$(echo "$GOROOT" | cut -c1)" != "/" ]; then
  error_and_abort "\$GOROOT must be an absolute path but it is set to $GOROOT"
fi

error_and_abort() {
  printf '\n  %s: %s\n\n' "ERROR" "$*" >&2
  exit 1
}

print_banner() {
    cat <<-'EOF'
=================================================
              ____
             / ___|_ __ ___   ___
            | |   | '__/ _ \ / __|
            | |___| | | (_) | (__
             \____|_|  \___/ \___|

       ___           _        _ _
      |_ _|_ __  ___| |_ __ _| | | ___ _ __
       | || '_ \/ __| __/ _` | | |/ _ \ '__|
       | || | | \__ \ || (_| | | |  __/ |
      |___|_| |_|___/\__\__,_|_|_|\___|_|
==================================================
EOF
}


print_message() {
  local message
  local severity
  local red
  local green
  local yellow
  local nc

  message="${1}"
  severity="${2}"
  red='\e[0;31m'
  green='\e[0;32m'
  yellow='\e[1;33m'
  nc='\e[0m'

  case "${severity}" in
    "info" ) echo -e "${nc}${message}${nc}";;
      "ok" ) echo -e "${green}${message}${nc}";;
   "error" ) echo -e "${red}${message}${nc}";;
    "warn" ) echo -e "${yellow}${message}${nc}";;
  esac
}

get_os() {
  local uname_out
  if command -v uname >/dev/null 2>&1; then
    uname_out="$(uname)"
    if [[ "${uname_out}" == "" ]]; then
      return 1
    else
      echo "${uname_out}"
      return 0
    fi
  else
    return 20
  fi
}

get_arch () {
    local arch=""
    local arch_check=${ASDF_GOLANG_OVERWRITE_ARCH:-"$(uname -m)"}
    case "${arch_check}" in
        x86_64|amd64) arch="amd64"; ;;
        i686|i386|386) arch="386"; ;;
        armv6l|armv7l) arch="armv6l"; ;;
        aarch64|arm64) arch="arm64"; ;;
        ppc64le) arch="ppc64le"; ;;
        *)
            fail "Arch '${arch_check}' not supported!"
            ;;
    esac
    printf "%s" "$arch"
}

get_platform () {
    local silent=${1:-}
    local platform=""
    platform="$(uname | tr '[:upper:]' '[:lower:]')"
    case "$platform" in
        linux|darwin|freebsd)
            [ -z "$silent" ] && msg "Platform '${platform}' supported!"
            ;;
        *)
            fail "Platform '${platform}' not supported!"
            ;;
    esac
    printf "%s" "$platform"
}

main() {
  local release="1.0.0"
  local os="$(uname -s | awk '{print tolower($0)}')"
  local oss=`get_os|tr "[A-Z]" "[a-z]"`
  print_banner
  echo $oss
}

main "$@" || exit 1