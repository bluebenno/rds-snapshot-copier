# README #

## RDS-snapshot-copier ###

Copy tagged AWS RDS Snapshots from region-a to region-b. Handles permutations of encrypted and unencryped.

## Config ##

Configuration is via environmental variables.

- The app runs in an infinite loop, with an hours sleep at the end of each loop. Override `RUN_EVERY_MINS`
- AWS RDS Snapshots are located in `SOURCE_REGION`. These will be copied to `TARGET_REGION`
- "in scope" are determined by the RDS being availabile and it's snapshot having a tag of `COPYTO`
- Optinal: Snapshots in the target region may be (re)encrypted using the RDS KMS key `TARGET_KMS`
- Optional: Snapshots maybe target region are housekept. Only the latest `MAX_SNAPSHOT_TGT` will be kept.
