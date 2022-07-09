#!/bin/sh

# Pretty Go Version Manager
#
# https://github.com/voocel/sv
#
# voocel, voocel@gmail.com

set -eu

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

get_arch() {
    local arch=""
    local arch_check=${ASDF_GOLANG_OVERWRITE_ARCH:-"$(uname -m)"}
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

get_platform () {
    local silent=${1:-}
    local platform=""
    platform="$(uname | tr '[:upper:]' '[:lower:]')"
    case "$platform" in
        linux|darwin|freebsd)
            [ -z "$silent" ] && msg "Platform '${platform}' supported!"
            ;;
        *)
            print_error "Platform '${platform}' not supported!"
            ;;
    esac
    printf "%s" "$platform"
}

get_shell_profile() {
    if [ -n "$($SHELL -c 'echo $ZSH_VERSION')" ]; then
        shell_profile="$HOME/.zshrc"
    elif [ -n "$($SHELL -c 'echo $BASH_VERSION')" ]; then
        shell_profile="$HOME/.bashrc"
    fi
}

set_env() {
cat>"$HOME/.sv/env"<<EOF
#!/bin/sh
# sv shell setup
case ":${PATH}:" in
    *:"$HOME/.sv/go/bin":*)
        ;;
    *)
        export PATH="$HOME/.sv/go:$PATH"
        export PATH="$HOME/.sv/bin:$PATH"
        ;;
esac
EOF
}

init_env() {
    local envStr='. "$HOME/.sv/env"'
    if grep -q "$envStr" "${shell_profile}"; then
        echo "${BLUE}SV env has exists in $shell_profile${RESET}"
    else
        echo $envStr >> "${shell_profile}"
    fi

    . ${shell_profile}
    source ${shell_profile}
}

check_curl() {
    if !(test -x "$(command -v curl)"); then
        print_error "You must pre-install the curl tool"
    fi
}

get_sv_bin() {
    SV_BIN=''
    THISOS=$(uname -s)
    ARCH=$(uname -m)

    case $THISOS in
       Linux*)
          case $ARCH in
            arm64)
              SV_BIN="sv-linux-arm-64"
              ;;
            aarch64)
              SV_BIN="sv-linux-arm-64"
              ;;
            *)
              SV_BIN="sv-linux-amd-64"
              ;;
          esac
          ;;
       Darwin*)
          case $ARCH in
            arm64)
              SV_BIN="sv-darwin-arm-64"
              ;;
            *)
              SV_BIN="sv-darwin-64"
              ;;
          esac
          ;;
       Windows*)
          SV_BIN="sv-windows-64.exe"
          ;;
    esac
}

get_latest_tag() {
    release=$(curl -s "https://api.github.com/repos/voocel/sv/releases/latest" | grep '"tag_name":' | cut -d'"' -f4)
}

main() {
    setup_color
    echo "${YELLOW}[1/3] Get sv latest version${RESET}"
    get_latest_tag
    if [ -z "$release" ]; then
        print_error "Get sv latest version error"
    fi
    printf "${GREEN}The sv latest version is %s${RESET}\n" $release

    # local os="$(uname -s | awk '{print tolower($0)}')"
    local os=`get_os|tr "[A-Z]" "[a-z]"`
    print_banner
    echo $os

    if [ -f ~/.bash_profile ]; then
        . ~/.bash_profile
    fi

    echo "${YELLOW}[2/3] Downloading sv to the /usr/local/bin${RESET}"
    check_curl
    get_sv_bin

    if [ ! -d $HOME/.sv/bin ];then
        mkdir -p $HOME/.sv/bin
    fi
    echo https://github.com/voocel/sv/releases/download/$release/$SV_BIN
    curl -kLs https://github.com/voocel/sv/releases/download/$release/$SV_BIN -o $HOME/.sv/bin/sv
    chmod +x $HOME/.sv/bin/sv
    echo "${GREEN}Installed successfully to: $HOME/.sv/bin/sv${RESET}"

    echo "${YELLOW}[3/3] Setting environment variables${RESET}"
    set_env
    get_shell_profile
    echo ${shell_profile}

    init_env
    echo "${GREEN}Set env successfully${RESET}"


#  [ -z "$GOROOT" ] && GOROOT="$HOME/.go"
#  [ -z "$GOPATH" ] && GOPATH="$HOME/go"
#  mkdir -p "$GOPATH"/{src,pkg,bin} "$GOROOT"

}

main "$@" || exit 1