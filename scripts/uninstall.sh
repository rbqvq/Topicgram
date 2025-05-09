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

service=""

print_cyan "$PROGRAM uninstallation script"

# Read parameters
{
    while [ $# -gt 0 ]; do
        case $1 in
        --service)
            service=$2
            shift
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
            print_red "No installed program found!"
            exit 1
        fi

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
            print_white " Detected ${#DIRS[@]} installed directions, please select one:"
            for DIR in ${DIRS[@]}; do
                print_white " - $DIR"
            done

            while [ -z "$service" ] || [ ! -f "/opt/$service/$PROGRAM" ]; do
                [ -n "$service" ] && print_red " Wrong input!"

                read -ep " Please type directions you want to uninstall (you can type 'all' for all of listed directions): " service

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

# Stop program
{
    print_yellow " ** Stopping program..."

    for SERVICE in ${SERVICES[@]}; do
        if [ -f "/etc/systemd/system/$SERVICE.service" ]; then
            systemctl disable --now $SERVICE
        else
            print_red " Service file '/etc/systemd/system/$SERVICE.service' not found, you need stop program by yourself!"
        fi
    done
}

# Remove files
{
    print_yellow " ** Removing files..."

    for SERVICE in ${SERVICES[@]}; do
        rm -rf /opt/$SERVICE
        rm -f /etc/systemd/system/$SERVICE.service
    done
}

# Finish uninstallation
{
    systemctl daemon-reload
}

print_green "$PROGRAM uninstalled successfully"
