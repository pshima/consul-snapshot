# consul-snapshot

consul-snapshot is a backup utility for Consul (https://www.consul.io).  This is slightly different than some other utilities out there as this runs as a daemon.  This is intended to run under Nomad (https://www.nomadproject.io).

WARNING: consul-snapshot is still in early development use at your own risk.  This is not used in production.

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
consul-snapshot backup
```

Running a restore:
```
consul-snapshot restore path/to/file/in/s3/bucket
```

## Todos
- Add unit tests
- Inspect app performance on larger data structures
- Add consul health checks
- Backup in chunks instead of all at once
- Add a web interface to view backups
- Add metrics
- Add single key backups
- Add options to specify paths
- Register as a consul service with health checks on last backup time
