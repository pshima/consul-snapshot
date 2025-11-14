# consul-snapshot nomad jobfile
job "consul-snapshot" {
  datacenters = [ "us-east-1" ]
  region      = "us"
  type = "service"

  update {
    max_parallel = 1
    min_healthy_time = "30s"
    healthy_deadline = "10m"
    auto_revert = true
    canary = 1
  }

  group "consul-snapshot" {
    count = 1

    task "backup" {
      driver = "docker"
      config {
        image = "consul-snapshot"
        args = ["backup"]
        port_map = {
          http = 5001
        }
      }

      service {
        port = "http"
        name = "consul-snapshot"
        check {
          type = "http"
          path = "/health"
          interval = "10s"
          timeout = "2s"
        }
      }

      env {
        "S3BUCKET" = "backups.example.bucket"
        "S3REGION" = "us-east-1"
        # "S3ENDPOINT" = "https://minio.example.com:9000"  # Optional: for S3-compatible services
        "BACKUPINTERVAL" = 300
        "CONSUL_HTTP_ADDR" = "consul.example.com:8500"
      }

      resources {
        cpu = 100
        memory = 256
        network {
          mbits = 100
          port "http" {
          }
        }
      }

    }
  }
}
