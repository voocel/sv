#!/bin/sh

# Pretty Go Version Manager
#
# https://github.com/voocel/sv
#
# voocel, voocel@gmail.com

set -eu

GOPATH=${GOPATH:-$HOME/go}
GOROOT=${GOROOT:-$HOME/.sv/go}

if [ "$(echo "$GOROOT" | cut -c1)" != "/" ]; then
    print_error "$GOROOT must be an absolute path but it is set to $GOROOT"
fi

print_banner() {
    cat <<-'EOF'
=================================================
              ___
             /  /\          ___
            /  /::\        /  /\
           /__/:/\:\      /  /:/
          _\_ \:\ \:\    /  /:/
         /__/\ \:\ \:\  /__/:/  ___
         \  \:\ \:\_\/  |  |:| /  /\
          \  \:\_\:\    |  |:|/  /:/
           \  \:\/:/    |__|:|__/:/
            \  \::/      \__\::::/
             \__\/           ````
       ___           _        _ _
      |_ _|_ __  ___| |_ __ _| | | ___ _ __
       | || '_ \/ __| __/ _` | | |/ _ \ '__|
       | || | | \__ \ || (_| | | |  __/ |
      |___|_| |_|___/\__\__,_|_|_|\___|_|
==================================================
EOF
}

setup_color() {
    RESET=$(printf '\033[0m')
    BOLD=$(printf '\033[1m')
    RED=$(printf '\033[31m')
    GREEN=$(printf '\033[32m')
    YELLOW=$(printf '\033[33m')
    BLUE=$(printf '\033[34m')
    PINK=$(printf '\033[35m')
    CYAN=$(printf '\033[36m')
    GRAY=$(printf '\033[37m')
}

print_error() {
    printf '%sError: %s%s\n' "${BOLD}${RED}" "$*" "${RESET}" >&2
    exit 1
}

get_os() {
    uname_out=""
    if command -v uname >/dev/null 2>&1; then
        uname_out="$(uname)"
        if [ "${uname_out}" = "" ]; then
            return 1
        else
            echo "${uname_out}"
            return 0
        fi
    else
        return 20
    fi
}

get_arch() {
    arch=""
    # silent=${1:-}
    arch_check=${ASDF_GOLANG_OVERWRITE_ARCH:-"$(uname -m)"}
    case "${arch_check}" in
        x86_64|amd64) arch="amd64"; ;;
        i686|i386|386) arch="386"; ;;
        armv6l|armv7l) arch="armv6l"; ;;
        aarch64|arm64) arch="arm64"; ;;
        ppc64le) arch="ppc64le"; ;;
        *)
            print_error "Arch '${arch_check}' not supported!"
            ;;
    esac
    printf "%s" "$arch"
}

get_shell_profile() {
    shell_profile=""
    if [[ "${SHELL}" == *"bash"* ]]; then
      if [[ -f "$HOME/.bashrc" ]]; then
        shell_profile="$HOME/.bashrc"
      elif [[ -f "$HOME/.bash_profile" ]]; then
        shell_profile="$HOME/.bash_profile"
      fi
    elif [[ "${SHELL}" == *"zsh"* ]]; then
      shell_profile="$HOME/.zshrc"
    fi

    if [ -z "$shell_profile" ]; then
       print_error "Get shell profile error, please set .bashrc"
    fi
}

gen_env() {
cat>"$HOME/.sv/env"<<EOF
#!/bin/sh
# sv shell setup
case ":${PATH}:" in
    *:"$HOME/.sv/go/bin:$HOME/.sv/bin":*)
        ;;
    *)
        export GO111MODULE=auto
        export SVHOME=$HOME/.sv
        export GOROOT=$HOME/.sv/go
        export GOPROXY=https://goproxy.cn,direct
        export PATH="$HOME/.sv/go/bin:$HOME/.sv/bin:$PATH"
        ;;
esac
EOF
}

set_env() {
    envStr='. "$HOME/.sv/env"'
    if grep -q "$envStr" "${shell_profile}"; then
        echo "${BLUE}SV env has exists in $shell_profile${RESET}"
    else
        echo $envStr >> "${shell_profile}"
    fi

    # . ${shell_profile}
    # source $shell_profile
}

## Detect the curl
check_curl() {
    if ! (test -x "$(command -v curl)"); then
        print_error "You must pre-install the curl tool"
    fi
}

get_svbin() {
    svbin=''
    THISOS=$(uname -s)
    ARCH=$(uname -m)

    case $THISOS in
       Linux*)
          case $ARCH in
            arm64)
              svbin="sv-linux-arm-64"
              ;;
            aarch64)
              svbin="sv-linux-arm-64"
              ;;
            *)
              svbin="sv-linux-amd-64"
              ;;
          esac
          ;;
       Darwin*)
          case $ARCH in
            arm64)
              svbin="sv-darwin-arm-64"
              ;;
            *)
              svbin="sv-darwin-64"
              ;;
          esac
          ;;
       Windows*)
          svbin="sv-windows-64.exe"
          ;;
    esac
}

get_latest_tag() {
    local release=$(curl -s "https://api.github.com/repos/voocel/sv/releases/latest" | grep '"tag_name":' | cut -d'"' -f4)
    echo "$release"
}

main() {
    setup_color
    echo "${YELLOW}[1/3] Get sv latest version${RESET}"
    local release="$(get_latest_tag)"
    if [ -z "$release" ]; then
        print_error "Get sv latest version error, please try again"
    fi
    printf "${GREEN}The sv latest version is %s${RESET}\n" $release

    # local os="$(uname -s | awk '{print tolower($0)}')"
    os=`get_os|tr "[A-Z]" "[a-z]"`
    print_banner

    if [ -f "$HOME/.bash_profile" ]; then
        . "$HOME/.bash_profile"
    fi

    echo "${YELLOW}[2/3] Downloading sv to the /usr/local/bin${RESET}"
    check_curl
    get_svbin

    if [ ! -d "$HOME/.sv/bin" ];then
        mkdir -p "$HOME/.sv/bin"
    fi

    download_url=https://github.com/voocel/sv/releases/download/$release/$svbin
    echo download_url
    http_code=$(curl -I -w '%{http_code}' -s -o /dev/null "$download_url")
    if [ "$http_code" -eq 404 ] || [ "$http_code" -eq 403 ]; then
        print_error "URL: ${download_url} returned status ${http_code}"
    fi
    curl -kLs download_url -o "$HOME/.sv/bin/sv"
    chmod +x "$HOME/.sv/bin/sv"
    echo "${GREEN}Installed successfully to: $HOME/.sv/bin/sv${RESET}"

    echo "${YELLOW}[3/3] Setting environment variables${RESET}"
    gen_env
    get_shell_profile
    set_env
    echo "${GREEN}Set env successfully${RESET}"

#  [ -z "$GOROOT" ] && GOROOT="$HOME/.go"
#  [ -z "$GOPATH" ] && GOPATH="$HOME/go"
#  mkdir -p "$GOPATH"/{src,pkg,bin} "$GOROOT"

}

set +u
if [ -z "${SVHOME}" ]; then
    set -u
    main "$@" || exit 1
else
    echo "${YELLOW}Already installed!${RESET}"
    echo "${YELLOW}You can delete the directory($HOME/.sv) and reinstall${RESET}"
fi
echo "${GREEN}end.${RESET}"
set -u