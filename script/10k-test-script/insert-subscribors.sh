#!/bin/bash

########################################################
# Script for inserting n subscribors
#
# Usage:
#   ./insert-subscribors.sh <n>
#
# Description:
#   This script is used to insert n subscribors.
########################################################

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

FREE5GC_CONSOLE_BASE_URL='http://127.0.0.1:5000'

FREE5GC_CONSOLE_LOGIN_DATA_FILE="${SCRIPT_DIR}/free5gc-console-login-data.json"
FREE5GC_CONSOLE_SUBSCRIBER_DATA_FILE="${SCRIPT_DIR}/free5gc-console-subscriber-data.json"

Usage() {
    echo "Usage: $0 <n>"
    exit 1
}

free5gc_console_login() {
    local token=$(curl -s -X POST $FREE5GC_CONSOLE_BASE_URL/api/login -H "Content-Type: application/json" -d @$FREE5GC_CONSOLE_LOGIN_DATA_FILE | jq -r '.access_token')
    if [ -z "$token" ] || [ "$token" = "null" ]; then
        echo "Failed to get token!"
        return 1
    fi

    echo "$token"
    return 0
}

main() {
    if [ $# -ne 1 ]; then
        Usage
    fi

    local n=$1

    local token=$(free5gc_console_login)
    if [ -z "$token" ]; then
        echo "Failed to get token!"
        return 1
    fi

    for i in $(seq 1 $n); do
        local imsi=$(jq -r '.ueId' "$FREE5GC_CONSOLE_SUBSCRIBER_DATA_FILE" | sed 's/imsi-//')
        local plmn_id=$(jq -r '.plmnID' "$FREE5GC_CONSOLE_SUBSCRIBER_DATA_FILE")

        if curl -s --fail -X POST $FREE5GC_CONSOLE_BASE_URL/api/subscriber/imsi-$imsi/$plmn_id -H "Content-Type: application/json" -H "Token: $token" -d @$FREE5GC_CONSOLE_SUBSCRIBER_DATA_FILE; then
            echo "Subscriber created successfully!"
        else
            echo "Failed to create subscriber!"
        fi

        local new_imsi=$((imsi + 1))
        jq --tab --arg new_ue_id "imsi-$new_imsi" '.ueId = $new_ue_id' "$FREE5GC_CONSOLE_SUBSCRIBER_DATA_FILE" > "${FREE5GC_CONSOLE_SUBSCRIBER_DATA_FILE}.tmp" && mv "${FREE5GC_CONSOLE_SUBSCRIBER_DATA_FILE}.tmp" "$FREE5GC_CONSOLE_SUBSCRIBER_DATA_FILE"
    done

    jq --tab '.ueId = "imsi-208930000000001"' "$FREE5GC_CONSOLE_SUBSCRIBER_DATA_FILE" > "${FREE5GC_CONSOLE_SUBSCRIBER_DATA_FILE}.tmp" && mv "${FREE5GC_CONSOLE_SUBSCRIBER_DATA_FILE}.tmp" "$FREE5GC_CONSOLE_SUBSCRIBER_DATA_FILE"
}

main "$@"