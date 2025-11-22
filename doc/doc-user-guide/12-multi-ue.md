# Multiple UEs

> [!Note]
> **Multiple UEs** means you can launch more than one UE instance with a single command.
>
> Before using the multiple UEs command, please refer to [free-ran-ue](02-free-ran-ue.md) or [free-ran-ue with Namespace](03-quickstart-free-ran-ue.md) for basic usage.

## Insert Subscribers

Before starting UEs, you need to go to free5GC's Webconsole and create subscribers.

free-ran-ue also provides insertion scripts under the `script/10k-test-script` directory for quickly inserting subscribers:

```bash
./script/10k-test-script/insert-subscribors.sh 1000 # here we use 1000 UEs as an example## UE Command Usage
```

## Start Multiple UEs

The `ue` command provides parameters for launching multiple UEs at once:

```bash
Usage:
  free-ran-ue ue [flags]

Examples:
  free-ran-ue ue

Flags:
  -p, --concurrent int   max concurrent UEs to start simultaneously (default 10)
  -c, --config string    config file path (default "config/ue.yaml")
  -h, --help             help for ue
  -n, --num int          number of UEs (default 1)
```

- `-p`, `--concurrent`: maximum number of UEs to start concurrently (default 10)
- `-n`, `--num`: total number of UEs to start (default 1)

If you want to launch 1000 UEs with up to 25 UEs starting concurrently:

```bash
./build/free-ran-ue ue -c config/ue.yaml -n 1000 -p 25
```

> [!Tip]
> The `-p` parameter should be tuned according to your machine's capacity to achieve faster and more stable launches.
