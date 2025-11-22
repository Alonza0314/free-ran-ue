#!/bin/bash

########################################################
# Script for integration test
#
# Usage:
#   ./integration-test.sh [basic | dc-static | dc-dynamic | ulcl]
#
# Description:
#   This script is used to test the functionality of free-ran-ue.
########################################################

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

DOCKER_PATH='../../docker'
BASIC_COMPOSE_FILE="${SCRIPT_DIR}/${DOCKER_PATH}/docker-compose.yaml"
DC_STATIC_COMPOSE_FILE="${SCRIPT_DIR}/${DOCKER_PATH}/docker-compose-dc-static.yaml"
DC_DYNAMIC_COMPOSE_FILE="${SCRIPT_DIR}/${DOCKER_PATH}/docker-compose-dc-dynamic.yaml"
ULCL_COMPOSE_FILE="${SCRIPT_DIR}/${DOCKER_PATH}/docker-compose-ulcl.yaml"

FREE5GC_CONSOLE_BASE_URL='http://127.0.0.1:5000'

FREE5GC_CONSOLE_LOGIN_DATA_FILE="${SCRIPT_DIR}/free5gc-console-login-data.json"
FREE5GC_CONSOLE_SUBSCRIBER_DATA_FILE="${SCRIPT_DIR}/free5gc-console-subscriber-data.json"

FRU_CONSOLE_BASE_URL='http://127.0.0.1:40104'

FRU_CONSOLE_LOGIN_DATA_FILE="${SCRIPT_DIR}/fru-console-login-data.json"
FRU_CONSOLE_UE_DC_SWITCH_DATA_FILE="${SCRIPT_DIR}/fru-console-ue-dc-switch-data.json"

TEST_POOL="basic|dc-static|dc-dynamic|ulcl"

Usage() {
    echo "Usage: $0 [basic | dc-static | dc-dynamic | ulcl]"
    exit 1
}

start_docker_compose() {
    if ! docker compose -f $1 up -d --wait --wait-timeout 180; then
        echo "Failed to start docker compose!"
        return 1
    fi
    sleep 2

    docker ps -a
    return 0
}

stop_docker_compose() {
    if ! docker compose -f $1 down; then
        echo "Failed to stop docker compose!"
        return 1
    fi

    return 0
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

free5gc_console_subscriber_action() {
    local token=$(free5gc_console_login)
    if [ -z "$token" ]; then
        echo "Failed to get token!"
        return 1
    fi

    local imsi=$(jq -r '.ueId' "$FREE5GC_CONSOLE_SUBSCRIBER_DATA_FILE" | sed 's/imsi-//')
    local plmn_id=$(jq -r '.plmnID' "$FREE5GC_CONSOLE_SUBSCRIBER_DATA_FILE")

    case $1 in
        "post")
            if curl -s --fail -X POST $FREE5GC_CONSOLE_BASE_URL/api/subscriber/imsi-$imsi/$plmn_id -H "Content-Type: application/json" -H "Token: $token" -d @$FREE5GC_CONSOLE_SUBSCRIBER_DATA_FILE; then
                echo "Subscriber created successfully!"
                return 0
            else
                echo "Failed to create subscriber!"
                return 1
            fi
        ;;
        "delete")
            if curl -s --fail -X DELETE $FREE5GC_CONSOLE_BASE_URL/api/subscriber/imsi-$imsi/$plmn_id -H "Content-Type: application/json" -H "Token: $token" -d @$FREE5GC_CONSOLE_SUBSCRIBER_DATA_FILE; then
                echo "Subscriber deleted successfully!"
                return 0
            else
                echo "Failed to delete subscriber!"
                return 1
            fi
        ;;
    esac
}

fru_console_login() {
    local token=$(curl -s -X POST $FRU_CONSOLE_BASE_URL/api/console/login -H "Content-Type: application/json" -d @$FRU_CONSOLE_LOGIN_DATA_FILE | jq -r '.token')
    if [ -z "$token" ] || [ "$token" = "null" ]; then
        echo "Failed to get token!"
        return 1
    fi

    echo "$token"
    return 0
}

fru_console_ue_dc_switch() {
    local token=$(fru_console_login)
    if [ -z "$token" ]; then
        echo "Failed to get token!"
        return 1
    fi

    if curl -s --fail -X POST $FRU_CONSOLE_BASE_URL/api/console/gnb/ue/nrdc -H "Content-Type: application/json" -H "Authorization: $token" -d @$FRU_CONSOLE_UE_DC_SWITCH_DATA_FILE; then
        echo "UE DC switch successful!"
        return 0
    else
        echo "Failed to switch UE DC!"
        return 1
    fi
}

main() {
    if [[ ! "$TEST_POOL" =~ "$1" ]]; then
        echo "Invalid test type: $1"
        Usage
        exit 1
    fi

    case $1 in
        "basic")
            if ! start_docker_compose $BASIC_COMPOSE_FILE; then
                echo "Failed to start docker compose!"
                exit 1
            fi

            if ! free5gc_console_subscriber_action "post"; then
                echo "Failed to create subscriber!"
                stop_docker_compose $BASIC_COMPOSE_FILE
                exit 1
            fi

            docker exec -d ue ./free-ran-ue ue -c uecfg.yaml

            sleep 3

            if ! docker exec ue ping -I ueTun0 8.8.8.8 -c 5; then
                echo "Failed to ping 8.8.8.8!"
                free5gc_console_subscriber_action "delete"
                stop_docker_compose $BASIC_COMPOSE_FILE
                exit 1
            fi

            if ! free5gc_console_subscriber_action "delete"; then
                echo "Failed to delete subscriber!"
                stop_docker_compose $BASIC_COMPOSE_FILE
                exit 1
            fi

            if ! stop_docker_compose $BASIC_COMPOSE_FILE; then
                echo "Failed to stop docker compose!"
                exit 1
            fi
        ;;
        "dc-static")
            if ! start_docker_compose $DC_STATIC_COMPOSE_FILE; then
                echo "Failed to start docker compose!"
                exit 1
            fi

            if ! free5gc_console_subscriber_action "post"; then
                echo "Failed to create subscriber!"
                stop_docker_compose $DC_STATIC_COMPOSE_FILE
                exit 1
            fi

            docker exec -d ue ./free-ran-ue ue -c uecfg.yaml

            sleep 3

            if ! docker exec ue ping -I ueTun0 8.8.8.8 -c 5; then
                echo "Failed to ping 8.8.8.8!"
                free5gc_console_subscriber_action "delete"
                stop_docker_compose $DC_STATIC_COMPOSE_FILE
                exit 1
            fi

            if ! docker exec ue ping -I ueTun0 1.1.1.1 -c 5; then
                echo "Failed to ping 8.8.8.8!"
                free5gc_console_subscriber_action "delete"
                stop_docker_compose $DC_STATIC_COMPOSE_FILE
                exit 1
            fi

            if ! free5gc_console_subscriber_action "delete"; then
                echo "Failed to delete subscriber!"
                stop_docker_compose $DC_STATIC_COMPOSE_FILE
                exit 1
            fi

            if ! stop_docker_compose $DC_STATIC_COMPOSE_FILE; then
                echo "Failed to stop docker compose!"
                exit 1
            fi
        ;;
        "dc-dynamic")
            if ! start_docker_compose $DC_DYNAMIC_COMPOSE_FILE; then
                echo "Failed to start docker compose!"
                exit 1
            fi

            if ! free5gc_console_subscriber_action "post"; then
                echo "Failed to create subscriber!"
                stop_docker_compose $DC_DYNAMIC_COMPOSE_FILE
                exit 1
            fi

            docker exec -d ue ./free-ran-ue ue -c uecfg.yaml

            sleep 3

            if ! docker exec ue ping -I ueTun0 8.8.8.8 -c 5; then
                echo "Failed to ping 8.8.8.8!"
                free5gc_console_subscriber_action "delete"
                stop_docker_compose $DC_DYNAMIC_COMPOSE_FILE
                exit 1
            fi

            if ! docker exec ue ping -I ueTun0 1.1.1.1 -c 5; then
                echo "Failed to ping 1.1.1.1!"
                free5gc_console_subscriber_action "delete"
                stop_docker_compose $DC_DYNAMIC_COMPOSE_FILE
                exit 1
            fi

            if ! fru_console_ue_dc_switch; then
                echo "Failed to switch UE DC!"
                free5gc_console_subscriber_action "delete"
                stop_docker_compose $DC_DYNAMIC_COMPOSE_FILE
                exit 1
            fi

            sleep 3

            if ! docker exec ue ping -I ueTun0 8.8.8.8 -c 5; then
                echo "Failed to ping 8.8.8.8!"
                free5gc_console_subscriber_action "delete"
                stop_docker_compose $DC_DYNAMIC_COMPOSE_FILE
                exit 1
            fi

            if ! docker exec ue ping -I ueTun0 1.1.1.1 -c 5; then
                echo "Failed to ping 1.1.1.1!"
                free5gc_console_subscriber_action "delete"
                stop_docker_compose $DC_DYNAMIC_COMPOSE_FILE
                exit 1
            fi

            if ! fru_console_ue_dc_switch; then
                echo "Failed to switch UE DC!"
                free5gc_console_subscriber_action "delete"
                stop_docker_compose $DC_DYNAMIC_COMPOSE_FILE
                exit 1
            fi

            sleep 3

            if ! docker exec ue ping -I ueTun0 8.8.8.8 -c 5; then
                echo "Failed to ping 8.8.8.8!"
                free5gc_console_subscriber_action "delete"
                stop_docker_compose $DC_DYNAMIC_COMPOSE_FILE
                exit 1
            fi

            if ! docker exec ue ping -I ueTun0 1.1.1.1 -c 5; then
                echo "Failed to ping 1.1.1.1!"
                free5gc_console_subscriber_action "delete"
                stop_docker_compose $DC_DYNAMIC_COMPOSE_FILE
                exit 1
            fi

            if ! free5gc_console_subscriber_action "delete"; then
                echo "Failed to delete subscriber!"
                stop_docker_compose $DC_DYNAMIC_COMPOSE_FILE
                exit 1
            fi

            if ! stop_docker_compose $DC_DYNAMIC_COMPOSE_FILE; then
                echo "Failed to stop docker compose!"
                exit 1
            fi
        ;;
        "ulcl")
            if ! start_docker_compose $ULCL_COMPOSE_FILE; then
                echo "Failed to start docker compose!"
                exit 1
            fi

            if ! free5gc_console_subscriber_action "post"; then
                echo "Failed to create subscriber!"
                stop_docker_compose $ULCL_COMPOSE_FILE
                exit 1
            fi

            docker exec -d ue ./free-ran-ue ue -c uecfg.yaml

            sleep 3

            if ! docker exec ue ping -I ueTun0 8.8.8.8 -c 5; then
                echo "Failed to ping 8.8.8.8!"
                free5gc_console_subscriber_action "delete"
                stop_docker_compose $ULCL_COMPOSE_FILE
                exit 1
            fi

            if ! docker exec ue ping -I ueTun0 1.1.1.1 -c 5; then
                echo "Failed to ping 1.1.1.1!"
                free5gc_console_subscriber_action "delete"
                stop_docker_compose $ULCL_COMPOSE_FILE
                exit 1
            fi

            if ! free5gc_console_subscriber_action "delete"; then
                echo "Failed to delete subscriber!"
                stop_docker_compose $ULCL_COMPOSE_FILE
                exit 1
            fi

            if ! stop_docker_compose $ULCL_COMPOSE_FILE; then
                echo "Failed to stop docker compose!"
                exit 1
            fi
        ;;
        *)
            Usage
            exit 1
        ;;
    esac
}

main "$@"