# README #

## rds-snapshot-copier ###

Copy tagged AWS rds Snapshots from region-a to region-b. Handles permutations of encrypted and unencryped.

## Config ##

Configuration can be via environmental variables:

- The app runs in an infinite loop, with an hours sleep at the end of each loop. Override `RUN_EVERY_MINS`
- AWS rds Snapshots are located in `SOURCE_REGION`. Inscope ones will be copied to `TARGET_REGION`
- Inscope rds Snapshots are 'available' AND have an AWS tag _key_ of `COPYTO`
- Optional: Snapshots in the target region may be (re)encrypted using the rds KMS key `TARGET_KMS`
- Optional: Snapshots in the _target_ region can be housekept. Only the latest `MAX_SNAPSHOT_TGT` will be kept, the rest deleted
- Optional: `LOG_LEVEL` has default of info. "debug", "info", "warn", "error", "dpanic", "panic", and "fatal" are valid
- Optional: `MAX_SNAPSHOT_FLIGHT` has default of 2.  You can override it, bearing in mind AWS Maxium is six between regions
