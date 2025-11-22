#!/bin/bash

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

INTEGRATION_TEST_SCRIPT="${SCRIPT_DIR}/integration-test.sh"

Usage() {
    echo "Usage: $0 [basic | dc-static | dc-dynamic | ulcl | all]"
    exit 1
}

main() {
    case $1 in
        "basic")
            if ! $INTEGRATION_TEST_SCRIPT basic; then
                echo "Failed to run test basic!"
                exit 1
            fi
        ;;
        "dc-static")
            if ! $INTEGRATION_TEST_SCRIPT dc-static; then
                echo "Failed to run test dc-static!"
                exit 1
            fi
        ;;
        "dc-dynamic")
            if ! $INTEGRATION_TEST_SCRIPT dc-dynamic; then
                echo "Failed to run test dc-dynamic!"
                exit 1
            fi
        ;;
        "ulcl")
            if ! $INTEGRATION_TEST_SCRIPT ulcl; then
                echo "Failed to run test ulcl!"
                exit 1
            fi
        ;;
        "all")
            # Run test basic
            if ! $INTEGRATION_TEST_SCRIPT basic; then
                echo "Failed to run test basic!"
                exit 1
            fi

            # Run test dc-static
            if ! $INTEGRATION_TEST_SCRIPT dc-static; then
                echo "Failed to run test dc-static!"
                exit 1
            fi

            # Run test dc-dynamic
            if ! $INTEGRATION_TEST_SCRIPT dc-dynamic; then
                echo "Failed to run test dc-dynamic!"
                exit 1
            fi

            # Run test ulcl
            if ! $INTEGRATION_TEST_SCRIPT ulcl; then
                echo "Failed to run test ulcl!"
                exit 1
            fi

            echo "All tests passed!"
        ;;
        *)
            Usage
            exit 1
        ;;
    esac
}

main "$@"