# consul-snapshot

consul-snapshot is a backup utility for consul (http://consul.io).  This is slightly different than some other utilities out there as this runs as a daemon.  It also has support for single backups with multiple nodes utilizing consul locking.

consul-snapshot is still in early development and lacks critical features such as restore, use at your own risk.

## Configuration
Configuration is done from several keys within consul
- /service/consul-snapshot/enabled (determines if it should run if the key exists)
- /service/consul-snapshot/s3bucket (the s3 bucket to write to)
- /service/consul-snapshot/s3region (the s3 region the bucket is in)

## Authentication
This does not use any authentication in consul and uses environment variables for the aws-sdk-go S3 configuration.

## Running
Just run the binary, there are no options