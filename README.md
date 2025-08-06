# consul-snapshot [![](https://travis-ci.org/pshima/consul-snapshot.svg)](https://travis-ci.org/pshima/consul-snapshot)

consul-snapshot is a backup and restore utility for Consul (https://www.consul.io).  This is slightly different than some other utilities out there as this runs as a daemon for backups and ships them to S3.  consul snapshot in its current state is designed only for disaster recovery scenarios and full restore.  There is no support for single key or path based backups at the moment.

This is intended to run under Nomad (https://www.nomadproject.io) and connected to Consul (https://www.consul.io) and registered as a service with health checks.  It also runs fine outside of Nomad standalone and can even be used for single backups, however it is designed to run as a daemon.

consul-snapshot runs a small http server that can be used for consul health checks on backup state.  Right now if the backup is older than 1 hour it will return 500s to health check requests at /health making it easy for consul health checking.  There is no consul service registration as that is expected to be done in the nomad job spec or manually.

consul-snapshot has been used in production since February 2016.

[CHANGELOG](CHANGELOG.md)

## Features
- Back up K/V Store
- Back up ACLs
- Back up Prepared Queries (Consul 0.6.x)
- Store backups in Amazon S3 / Google Cloud Storage
- Restore backups directly from S3 / Google Cloud Storage
- AWS encrypted backups and restores with configurable passphrase
- Consul compatible health checks for age of last backup
- Configurable consul settings and backup interval
- EC2 IAM instance profile support(no credentials needed)

## Installation
Grab the binary from [Releases](https://github.com/pshima/consul-snapshot/releases)

consul-snapshot requires go 1.24+ to build.

With go get:
```
go get github.com/pshima/consul-snapshot
```

From source:
```
git clone https://github.com/pshima/consul-snapshot
cd consul-snapshot
make
make install
```

## Configuration
Configuration is done from environment variables.
- S3BUCKET (the s3 bucket where backups should be delivered)
- S3REGION (the region the s3 bucket is located)
- AWS_ACCESS_KEY_ID (the access key id used to access the bucket)
- AWS_SECRET_ACCESS_KEY (the secret key used to access the bucket)
- GCSBUCKET (the Google Cloud Storage bucket where backups should be delivered)
- BACKUPINTERVAL (how often you want the backup to run in seconds)
- CRYPTO_PASSWORD (sets a password for encrypting and decrypting backups)
- SNAPSHOT_TMP_DIR (sets the directory for temporary files, defaults to "/tmp")
- CONSUL_SNAPSHOT_UPLOAD_PREFIX (an arbitrary prefix to be prepended to the
  name of each uploaded object, e.g., `consul-dc1`.  Default is `backups`.)
- CONSUL_SNAPSHOT_S3_SSE (optional server-side encryption
  algorithm, e.g., `AES256` or `aws:kms`)
- CONSUL_SNAPSHOT_S3_SSE_KMS_KEY_ID (optional KMS key ID, if
  server-side encryption is used, and `aws:kms` is used for the
  encryption algorithm)

And through the consul api there are several options available (https://github.com/hashicorp/consul/blob/master/api/api.go#L126)

- CONSUL_HTTP_ADDR (default: 127.0.0.1:8500)
- CONSUL_HTTP_TOKEN (default: nil)
- CONSUL_HTTP_AUTH (default: nil)
- CONSUL_HTTP_SSL (default: nil)
- CONSUL_HTTP_SSL_VERIFY (default: nil)

## Authentication
Authentication is done through the above environment variables.  Credentials can be ommitted in place of an EC2 Instance IAM profile with write access to the S3 Bucket.

## Running
Running a backup:
```
% consul-snapshot backup
[INFO] v0.2.3: Starting Consul Snapshot
2017/08/16 09:33:25 [DEBUG] Backup starting on interval: 15s
2017/08/16 09:33:40 [INFO] Starting Backup At: 1502901220
2017/08/16 09:33:40 [INFO] Listing keys from consul
2017/08/16 09:33:40 [INFO] Converting 4 keys to JSON
2017/08/16 09:33:40 [INFO] Listing Prepared Queries from consul
2017/08/16 09:33:40 [INFO] Converting 0 keys to JSON
2017/08/16 09:33:40 [INFO] Listing ACLs from consul
2017/08/16 09:33:40 [INFO] ACL support detected as disbaled, skipping
2017/08/16 09:33:40 [INFO] Converting 0 ACLs to JSON
2017/08/16 09:33:40 [INFO] Preparing temporary directory for backup staging
2017/08/16 09:33:40 [INFO] Writing KVs to local backup file
2017/08/16 09:33:40 [DEBUG] Wrote 424 bytes to file, /tmp/macbook.local.consul.snapshot.1502901220/consul.kv.1502901220.json
2017/08/16 09:33:40 [INFO] Writing PQs to local backup file
2017/08/16 09:33:40 [DEBUG] Wrote 2 bytes to file, /tmp/macbook.local.consul.snapshot.1502901220/consul.pq.1502901220.json
2017/08/16 09:33:40 [INFO] Writing ACLs to local backup file
2017/08/16 09:33:40 [DEBUG] Wrote 2 bytes to file, /tmp/macbook.local.consul.snapshot.1502901220/consul.acl.1502901220.json
2017/08/16 09:33:40 [DEBUG] Wrote 339 bytes to file, /tmp/macbook.local.consul.snapshot.1502901220/meta.json
2017/08/16 09:33:40 [INFO] Writing Backup to Remote File
2017/08/16 09:33:40 [INFO] Uploading consul-backup-testing/backups/2017/8/16/macbook.local.consul.snapshot.1502901220.tar.gz to S3 in us-west-2
2017/08/16 09:33:40 [INFO] Running post processing
2017/08/16 09:33:40 [INFO] Backup completed successfully
```

Running a restore:
```
% consul-snapshot restore backups/2017/8/16/macbook.local.consul.snapshot.1502901220.tar.gz
[INFO] v0.2.3: Starting Consul Snapshot
2017/08/16 09:36:04 [DEBUG] Starting restore of consul-backup-testing/backups/2017/8/16/macbook.local.consul.snapshot.1502901220.tar.gz
2017/08/16 09:36:04 [INFO] Downloading consul-backup-testingbackups/2017/8/16/macbook.local.consul.snapshot.1502901220.tar.gz from S3 in us-west-2
2017/08/16 09:36:04 [INFO] Download completed
2017/08/16 09:36:04 [INFO] Checking encryption status of backup
2017/08/16 09:36:04 [INFO] Extracting backup
2017/08/16 09:36:04 [INFO] Inspecting backup contents
2017/08/16 09:36:04 [INFO] Found valid metadata of snapshot version 0.2.4 with unix_timestamp 1502901220
2017/08/16 09:36:04 [INFO] Parsing KV Data
2017/08/16 09:36:04 [INFO] Loaded 4 keys to restore
2017/08/16 09:36:04 [INFO] Parsing PQ Data
2017/08/16 09:36:04 [INFO] Loaded 0 Prepared Queries to restore
2017/08/16 09:36:04 [INFO] Parsing ACL Data
2017/08/16 09:36:04 [INFO] Loaded 0 ACLs to restore
2017/08/16 09:36:04 [INFO] Restored 4 keys with 0 errors
2017/08/16 09:36:04 [WARN] PQ restoration currently unsupported
2017/08/16 09:36:04 [WARN] ACL restoration currently unsupported
2017/08/16 09:36:04 [INFO] Restore completed.
```

## Testing

There are some unit tests but not near full coverage.

There is an acceptance test that:
- Spins up a local consul agent in dev mode
- Generates random k/v data and inserts it
- Takes a backup locally
- Wipes consul k/v
- Restores the backup
- Verifies that the k/v is still correct

To run the acceptance test set ACCEPTANCE_TEST=1

## Todos
- Add safety checks or confirm dialog for restore
- Add restore dry run
- Inspect app performance on larger data structures
- Backup in chunks instead of all at once
- Add a web interface to view backups
- Add metrics
- Add single key backups
- Use transactions for backups and restores
- Add support for just running once
- Add the ability to restore PQs and ACLs (currently back up only)
- Look at options for using consul's native snapshot commands
