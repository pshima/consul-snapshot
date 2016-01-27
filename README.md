# consul-snapshot

consul-snapshot is a backup utility for consul (http://consul.io).  This is slightly different than some other utilities out there as this runs as a daemon.  It also has support for single backups with multiple nodes utilizing consul locking.

consul-snapshot is still in early development and lacks critical features such as restore, use at your own risk.

## Configuration
Configuration is done from environment variables.
- S3BUCKET (the s3 bucket where backups should be delivered)
- S3REGION (the region the s3 bucket is located)
- AWS_ACCESS_KEY_ID (the access key id used to access the bucket)
- AWS_SECRET_ACCESS_KEY (the secret key used to access the bucket)
- BACKUPINTERVAL (how often you want the backup to run in seconds)

## Authentication
This does not use any authentication in consul and uses environment variables for the aws-sdk-go S3 configuration.

## Running
Just run the binary, there are no options, use the environment variables for configuration.

## Todo List
- Backup in chunks instead of all at once
- Add tests
- Add a web interface to view backups
- Add metrics
- Add single key backups
- Add options to specify paths
- Register as a consul service with health checks on last backup time
