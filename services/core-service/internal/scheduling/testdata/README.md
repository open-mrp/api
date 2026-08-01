# Parity fixtures

`merz_<date>_input.json` and `merz_<date>_expected.json` are captured from the TS
scheduling script and drive `parity_test.go`, which is the gate that proves the Go
solver reproduces the script's plan before anything persists one.

Capture with:

    cd dashboard/apps/api
    KNIT_DUMP_JSON=1 KNIT_GROWTH_MULT=1 bun run src/scripts/knit-scheduling-merz.ts

`KNIT_GROWTH_MULT=1` is required. The script's default doubles every demand bucket;
the Go port has no growth multiplier (that intent is expressed as a demand override),
so a fixture captured with the default would compare the solver against doubled demand
and fail for a reason unrelated to the port. `parity_test.go` refuses to run against a
fixture whose `growthMultiplier` is not 1.

The script is read-only against the database — it only writes these files.

`parity_test.go` skips when no fixture is present, because the capture needs
production data that CI does not have. A skip is not a pass.
