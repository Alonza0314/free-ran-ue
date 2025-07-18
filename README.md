# free-ran-ue

![free-ran-ue](/doc/image/free-ran-ue.jpg)

## Introduction

free-ran-ue is a tool designed to simulate interactions between the core network and RAN/UE. Its primary goal is to provide a practical platform for testing the [NR-DC(New Radio Dual Connectivity)](https://free5gc.org/blog/20250219/20250219/) feature.

For more details, please visit the [free-ran-ue official website](https://alonza0314.github.io/free-ran-ue/).

## Packages

free-ran-ue utilizes tool and model packages from [free5GC](https://github.com/free5gc).

## License

free-ran-ue is licensed under the [Apache 2.0](LICENSE) license.

Copyright © 2025 Alonza0314. All rights reserved.

## Log Description

There are five log levels available for both gNB and UE:

- ERROR: Critical errors that cause the application to stop.
- WARN: Unusual events that do not affect application functionality.
- INFO: General information that users should be aware of.
- DEBUG: Information useful for developers during debugging.
- TRACE: Detailed step-by-step information for in-depth analysis.

This can customized in the [configuration files](/config).
