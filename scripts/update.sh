#!/bin/bash
print_black() {
    echo -e "\033[30m$1\033[0m"
}

print_red() {
    echo -e "\033[31m$1\033[0m"
}

print_green() {
    echo -e "\033[32m$1\033[0m"
}

print_yellow() {
    echo -e "\033[33m$1\033[0m"
}

print_blue() {
    echo -e "\033[34m$1\033[0m"
}

print_magenta() {
    echo -e "\033[35m$1\033[0m"
}

print_cyan() {
    echo -e "\033[36m$1\033[0m"
}

print_grey() {
    echo -e "\033[37m$1\033[0m"
}

print_white() {
    echo "$1"
}

clear
PROGRAM="Topicgram"

PROJECT_OWNER="CoiaPrant"
PROJECT_NAME="Topicgram"
PROJECT_URL="https://gitlab.com/$PROJECT_OWNER/$PROJECT_NAME"
PROJECT_API_URL="https://gitlab.com/api/v4/projects/$PROJECT_OWNER%2F$PROJECT_NAME"

service=""
offline=0

print_cyan "$PROGRAM update script"

# Read parameters
{
    # Parse parameters
    while [ $# -gt 0 ]; do
        case $1 in
        --service)
            service=$2
            shift
            ;;
        --version)
            version=$2
            shift
            ;;
        --offline)
            offline=1
            ;;
        *)
            print_red " Unknown parameter: $1"
            exit 2
            ;;
        esac
        shift
    done
}

# Check system
{
    print_yellow " ** Checking system info..."

    # Check architecture
    case $(uname -m) in
    x86_64)
        arch="amd64"
        ;;
    armv7*)
        arch="armv7"
        ;;
    aarch64)
        arch="arm64"
        ;;
    s390x)
        arch="s390x"
        ;;
    *)
        print_red " Unsupported architecture"
        exit 1
        ;;
    esac

    # Check systemd
    command -V systemctl >/dev/null
    if [ "$?" -ne 0 ]; then
        print_red "Not found systemd"
        exit 1
    fi
}

# Check installed program
{
    print_yellow " ** Checking installation info..."

    SERVICES=()
    if [ -n "$service" ]; then
        if [ "$service" == "all" ]; then
            for DIR in $(ls /opt); do
                [ -f "/opt/$DIR/$PROGRAM" ] && DIRS=("${DIRS[@]}" "$DIR")
            done

            case ${#DIRS[@]} in
            0)
                print_red "No installed program found!"
                exit 1
                ;;
            1)
                service="${DIRS[0]}"

                SERVICES=("$service")
                print_white " Detected installed direction: /opt/$service"
                ;;
            *)
                print_white " Detected ${#DIRS[@]} installed directions:"
                for DIR in ${DIRS[@]}; do
                    print_white " - $DIR"
                done

                SERVICES=${DIRS[@]}
                ;;
            esac
        else
            if [ ! -d "/opt/$service" ]; then
                print_red "'/opt/$service' not exists!"
                exit 1
            fi

            if [ ! -f "/opt/$service/$PROGRAM" ]; then
                print_red "'/opt/$service' not include program!"
                exit 1
            fi

            SERVICES=("$service")
        fi
    else
        DIRS=()

        if [ ! -d "/opt" ]; then
            print_red "No installed program found, please run install script first!"
            exit 1
        fi

        for DIR in $(ls /opt); do
            [ -f "/opt/$DIR/$PROGRAM" ] && DIRS=("${DIRS[@]}" "$DIR")
        done

        case ${#DIRS[@]} in
        0)
            print_red "No installed program found, please run install script first!"
            exit 1
            ;;
        1)
            service="${DIRS[0]}"

            SERVICES=("$service")
            print_white " Detected installed direction: /opt/$service"
            ;;
        *)
            print_white " Detected ${#DIRS[@]} installed directions, please select one:"
            for DIR in ${DIRS[@]}; do
                print_white " - $DIR"
            done

            while [ -z "$service" ] || [ ! -f "/opt/$service/$PROGRAM" ]; do
                [ -n "$service" ] && print_red " Wrong input!"

                read -ep " Please type directions you want to update (you can type 'all' for all of listed directions): " service

                if [ "$service" == "all" ]; then
                    SERVICES=${DIRS[@]}
                    break
                else
                    SERVICES=("$service")
                fi
            done
            ;;
        esac
    fi
}

# Update program
{
    # Prepare program package
    if [ $offline -eq 0 ]; then
        # Get latest release version
        {
            print_yellow " ** Checking release info..."

            if [ -z "$version" ]; then
                version=$(curl -sL "$PROJECT_API_URL/releases" | grep "tag_name" | head -n 1 | awk -F ":" '{print $2}' | awk -F "," '{print $1}' | sed 's/\"//g;s/,//g;s/ //g' | awk -F "v" '{print $2}')
                if [ -z "$version" ]; then
                    print_red "Unable to get releases info"
                    exit 1
                fi

                print_white " Detected lastet verion: $version"
            else
                print_white " Use the specified verion: $version"
            fi
        }

        # Download release
        {
            print_yellow " ** Downloading release..."

            curl -L -o /tmp/$PROGRAM.tar.gz "$PROJECT_URL/-/releases/v$version/downloads/${PROGRAM}_${version}_linux_${arch}.tar.gz"
            if [ $? -ne 0 ] || [ ! -f "/tmp/$PROGRAM.tar.gz" ]; then
                print_red "Download failed"
                exit 1
            fi
        }
    else
        print_yellow " ** Offline installation..."

        [ -z "$version" ] && version="1.0.0"

        print_white " Please download backend package, rename it to '$PROGRAM.tar.gz' and upload it to /tmp"
        print_white " If you selected version '$version', please open '$PROJECT_URL/-/releases/v$version/downloads/${PROGRAM}_${version}_linux_${arch}.tar.gz' in your browser. Replace the version by yourself!"
        print_white ""

        read -ep " Press [Enter] to continue installation..."
        while [ ! -f "/tmp/$PROGRAM.tar.gz" ]; do
            print_red " File not exists!"
            print_white " Please upload '$PROGRAM.tar.gz' to /tmp"

            read -ep " Press [Enter] to continue installation..."
        done
    fi

    # Decompress package
    {
        TMP_DIR=$(mktemp -d)
        if [ -z "$TMP_DIR" ]; then
            TMP_DIR="/tmp/$PROGRAM"
            mkdir -p $TMP_DIR
        fi

        tar -xzf /tmp/$PROGRAM.tar.gz -C $TMP_DIR
        if [ $? -ne 0 ] || [ ! -f "$TMP_DIR/$PROGRAM" ]; then
            print_red "Decompression failed"

            rm -rf $TMP_DIR
            exit 1
        fi

        for SERVICE in ${SERVICES[@]}; do
            rm -f /opt/$SERVICE/$PROGRAM
            cp -f $TMP_DIR/$PROGRAM /opt/$SERVICE/$PROGRAM
        done

        rm -f /tmp/$PROGRAM.tar.gz
        rm -rf $TMP_DIR
    }

    # Configure program
    {
        for SERVICE in ${SERVICES[@]}; do
            chmod +x /opt/$SERVICE/$PROGRAM
        done
    }
}

# Finish update
{
    print_yellow " ** Starting program..."

    for SERVICE in ${SERVICES[@]}; do
        if [ -f "/etc/systemd/system/$SERVICE.service" ]; then
            systemctl restart $SERVICE
        else
            print_red " Service file '/etc/systemd/system/$SERVICE.service' not found, you need restart program by yourself!"
        fi
    done
}

print_green "$PROGRAM updated successfully"
