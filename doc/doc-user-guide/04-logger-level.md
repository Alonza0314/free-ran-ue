# Logger Level

For gNB, UE and console, there is a log level field in the configuration file:

```yaml
logger:
  level: "info" # error, warn, info, debug, trace, test
```

## Log Description

There are six log levels available for gNB, UE and console:

- error: Critical errors that cause the application to stop.
- warn: Unusual events that do not affect application functionality.
- info: General information that users should be aware of.
- debug: Information useful for developers during debugging.
- trace: Detailed step-by-step information for in-depth analysis.
- test: Logs for developer checking anywhere.

This can be customized in the [configuration files](https://github.com/free-ran-ue/free-ran-ue/tree/main/config).
