#!/bin/sh
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

service="$PROGRAM"
offline=0

print_cyan "$PROGRAM installation script"

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

    # Check parameters validity
    {
        if [ -z "$service" ]; then
            print_red " Parameter 'service' not found"
            exit 2
        fi
    }
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

    # Check installed
    while [ -f "/etc/systemd/system/$service.service" ] || [ -d "/opt/$service" ] || [ "$service" == "all" ]; do
        read -ep " Service '$service' exists or invalid, please enter a new service name: " service
    done
}

# Install program
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

        print_white " Please download package, rename it to '$PROGRAM.tar.gz' and upload it to /tmp"
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
        if [ $? -ne 0 ] || [ ! -f "$TMP_DIR/$PROGRAM" ] || [ ! -f "$TMP_DIR/systemd/$PROGRAM.service" ]; then
            print_red "Decompression failed"

            rm -rf $TMP_DIR
            exit 1
        fi

        mkdir -p /opt/$service

        cp -f $TMP_DIR/$PROGRAM /opt/$service/$PROGRAM
        [ -d "$TMP_DIR/examples" ] && cp -rf $TMP_DIR/examples /opt/$service/examples

        rm -f /tmp/$PROGRAM.tar.gz
        rm -rf $TMP_DIR
    }

    # Configure program
    {
        chmod +x /opt/$service/$PROGRAM

        print_yellow " Please write 'config.json' manually, then run 'systemctl enable --now $service'"
    }

    # Add system service
    {
        cat >/etc/systemd/system/$service.service <<EOF
[Unit]
Description=Topicgram
Documentation=https://gitlab.com/CoiaPrant/Topicgram/
After=network.target

[Service]
Type=simple
User=root
Restart=always
RestartSec=20s
TasksMax=infinity
LimitCPU=infinity
LimitFSIZE=infinity
LimitDATA=infinity
LimitSTACK=infinity
LimitCORE=infinity
LimitRSS=infinity
LimitNOFILE=infinity
LimitAS=infinity
LimitNPROC=infinity
LimitSIGPENDING=infinity
LimitMSGQUEUE=infinity
LimitRTTIME=infinity
WorkingDirectory=/opt/$service
ExecStart=/opt/$service/Topicgram --config config.json
EOF
    }
}

# Finish installation
{
    print_yellow " ** Finishing installation..."

    systemctl daemon-reload
}

print_green "$PROGRAM installed successfully"
