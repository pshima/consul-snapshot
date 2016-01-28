# consul-snapshot

consul-snapshot is a backup and restore utility for Consul (https://www.consul.io).  This is slightly different than some other utilities out there as this runs as a daemon for backups and ships them to S3.  consul snapshot in its current state is designed only for disaster recovery scenarios and full restore.  There is no support for single key or path based backups at the moment.

This is intended to run under Nomad (https://www.nomadproject.io) and connected to Consul (https://www.consul.io) and registered as a service with health checks.

consul-snapshot runs a small http server that can be used for consul health checks on backup state.  Right now if the backup is older than 1 hour it will return 500s to health check requests at /health making it easy for consul health checking.  There is no consul service registration as that is expected to be done in the nomad job spec.

WARNING: consul-snapshot is still in early development use at your own risk.  Do not use this in production.

## Installation
Grab the binary from [Releases](/releases)

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
- BACKUPINTERVAL (how often you want the backup to run in seconds)

And through the consul api there are several options available (https://github.com/hashicorp/consul/blob/master/api/api.go#L126)

- CONSUL_HTTP_ADDR (default: 127.0.0.1:8500)
- CONSUL_HTTP_TOKEN (default: nil)
- CONSUL_HTTP_AUTH (default: nil)
- CONSUL_HTTP_SSL (default: nil)
- CONSUL_HTTP_SSL_VERIFY (default: nil)

## Authentication
Authentication is done through the above environment variables.

## Running
Running a backup:
```
$ consul-snapshot backup
[INFO] v0.1.0: Starting Consul Snapshot
2016/01/27 06:07:18 [DEBUG] Backup starting on interval: 30s
2016/01/27 06:07:48 [INFO] Starting Backup At: 1453874868
2016/01/27 06:07:48 [INFO] Listing keys from consul
2016/01/27 06:07:48 [INFO] Converting keys to JSON
2016/01/27 06:07:48 [INFO] Writing Local Backup File
2016/01/27 06:07:48 [DEBUG] Wrote 37362 bytes to file, /tmp/consul.backup.1453874868.gz
2016/01/27 06:07:48 [INFO] Writing Backup to Remote File
2016/01/27 06:07:48 [INFO] Uploading testbucket/consul.backup.1453874868.gz to S3 in us-west-2
2016/01/27 06:07:48 [INFO] Running post processing
2016/01/27 06:08:18 [INFO] Starting Backup At: 1453874898
2016/01/27 06:08:18 [INFO] Listing keys from consul
2016/01/27 06:08:18 [INFO] Converting keys to JSON
2016/01/27 06:08:18 [INFO] Writing Local Backup File
2016/01/27 06:08:18 [DEBUG] Wrote 37362 bytes to file, /tmp/consul.backup.1453874898.gz
2016/01/27 06:08:18 [INFO] Writing Backup to Remote File
2016/01/27 06:08:18 [INFO] Uploading testbucket/consul.backup.1453874898.gz to S3 in us-west-2
2016/01/27 06:08:19 [INFO] Running post processing
```

Running a restore:
```
$ consul-snapshot restore consul.backup.1453928301.gz
[INFO] v0.1.0: Starting Consul Snapshot
2016/01/27 13:36:26 [DEBUG] Starting restore of testbucket/consul.backup.1453928301.gz
2016/01/27 13:36:26 [INFO] Downloading testbucket/tmp/consul.backup.1453928301.gz from S3 in us-west-2
2016/01/27 13:36:26 [INFO] Download completed
2016/01/27 13:36:26 [INFO] Extracting Backup File
2016/01/27 13:36:26 [INFO] Extracted 5 keys to restore
2016/01/27 13:36:26 [INFO] Restored 5 keys with 0 errors
2016/01/27 13:36:26 [INFO] Restore completed.
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
- Add more unit tests and fix acceptance testing logic to config itself
- Add more configurable options
- Add safety checks or confirm dialog for restore
- Add restore dry run
- Add checksumming/metadata on local file and upload meta first
- Inspect app performance on larger data structures
- Backup in chunks instead of all at once
- Add a web interface to view backups
- Add metrics
- Add single key backups
- Add options to specify paths
