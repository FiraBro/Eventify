# C1 execution status (uploaded archive review)

Status: **harness prepared; baseline not complete**.

| Requested measurement                         | Result in this environment                                     |
| --------------------------------------------- | -------------------------------------------------------------- |
| Availability p50/p95/p99, throughput, errors  | Not measured: no reachable configured API/test fixture/token   |
| Services-list p50/p95/p99, throughput, errors | Not measured: no reachable configured API                      |
| Settings p50/p95/p99, throughput, errors      | Not measured: no reachable configured API/test token           |
| PostgreSQL query counts per endpoint          | Unavailable: no database/container runtime in this environment |
| CPU/memory                                    | Unavailable: no API process/container running here             |
| Original commit / working-tree state          | Unavailable: uploaded ZIP excludes `.git`                      |

The only successful executions were `python3 -m py_compile scripts/performance/http_baseline.py` and `python3 scripts/performance/http_baseline.py --help`. The required Go verification commands were attempted and failed before compilation because Go tried to download toolchain `go1.26.9`, while network DNS access to `proxy.golang.org` is unavailable. `docker --version` also reported `docker: command not found`. `git diff --check` could not run because this extracted archive is not a Git repository.

Do not interpret missing measurements as zero or use them as a pre-cache baseline. Run the commands in `C1-baseline.md` in the actual checkout and save the generated JSON only after all requests succeed. Capture `git rev-parse HEAD` and `git status --short --branch` from the real checkout first.
