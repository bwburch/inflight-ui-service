# MinIO Setup for Workbench V2 Attachments

## Overview

Workbench V2 uses **MinIO** (S3-compatible object storage) for storing simulation attachments (screenshots, configs, logs).

**Why MinIO?**
- ✅ S3-compatible API (can swap to AWS S3 in production)
- ✅ Self-hosted and open-source
- ✅ Works in multi-server deployments
- ✅ Easy local development with Docker
- ✅ Built-in durability and versioning

---

## Quick Start

### 1. Start MinIO with Docker Compose

```bash
cd inflight-metrics-collector/documentation/docker-compose
docker-compose -f docker-compose.storage.yaml up -d minio
```

### 2. Verify MinIO is Running

```bash
# Check container status
docker ps | grep minio

# Health check
curl http://localhost:9010/minio/health/live

# Expected output: HTTP 200 OK
```

### 3. Access MinIO Console

**URL:** http://localhost:9011
**Credentials:**
- Username: `admin`
- Password: `admin_password`

### 4. Verify Bucket (Auto-Created)

The `inflight-simulations` bucket is automatically created when the UI service starts.

To verify manually:
1. Login to MinIO console
2. Navigate to "Buckets"
3. Look for `inflight-simulations`

---

## Configuration

### Current Configuration (Hardcoded)

**File:** `internal/api/server.go` (lines 60-66)

```go
minioEndpoint := "localhost:9010"
minioAccessKey := "admin"
minioSecretKey := "admin_password"
minioBucket := "inflight-simulations"
minioUseSSL := false
```

### TODO: Move to Config File

**Recommended:** `config/service.yaml`

```yaml
minio:
  endpoint: "localhost:9010"
  access_key: "admin"
  secret_key: "admin_password"
  bucket: "inflight-simulations"
  use_ssl: false
  region: "us-east-1"  # Optional, defaults to us-east-1
```

---

## Storage Structure

### S3 Key Pattern

```
simulations/{job_id}/{user_id}/{filename}
```

**Examples:**
```
simulations/142/5/screenshot-2025-10-07-1730.png
simulations/142/5/application.yaml
simulations/143/7/gc-log.txt
```

### Benefits:
- **Organized by job** - Easy to find all files for a job
- **User isolation** - Files grouped by uploader
- **Unique keys** - No collisions between users

### Bucket Policies

Default policy: **Private**
- Files accessible only via pre-signed URLs
- 1-hour expiration for download links
- No public read access

---

## API Integration

### Go SDK (Backend)

**Package:** `github.com/minio/minio-go/v7`

**Install:**
```bash
go get github.com/minio/minio-go/v7
```

**Usage:**
```go
import (
    "github.com/minio/minio-go/v7"
    "github.com/minio/minio-go/v7/pkg/credentials"
)

// Initialize client
client, err := minio.New("localhost:9010", &minio.Options{
    Creds:  credentials.NewStaticV4("admin", "admin_password", ""),
    Secure: false, // No SSL for local dev
})

// Upload file
uploadInfo, err := client.PutObject(ctx, "inflight-simulations", "simulations/142/5/file.png", fileReader, fileSize, minio.PutObjectOptions{
    ContentType: "image/png",
})

// Download file
object, err := client.GetObject(ctx, "inflight-simulations", "simulations/142/5/file.png", minio.GetObjectOptions{})
defer object.Close()

// Delete file
err = client.RemoveObject(ctx, "inflight-simulations", "simulations/142/5/file.png", minio.RemoveObjectOptions{})

// Generate presigned URL (1 hour expiration)
url, err := client.PresignedGetObject(ctx, "inflight-simulations", "simulations/142/5/file.png", time.Hour, nil)
```

---

## Production Deployment

### Option 1: Self-Hosted MinIO (Recommended for On-Prem)

**Docker Compose (Production):**
```yaml
minio:
  image: minio/minio:latest
  ports:
    - "9000:9000"
    - "9001:9001"
  volumes:
    - minio-data:/data
  environment:
    MINIO_ROOT_USER: ${MINIO_ROOT_USER}
    MINIO_ROOT_PASSWORD: ${MINIO_ROOT_PASSWORD}
  command: server /data --console-address ":9001"
  restart: always

volumes:
  minio-data:
    driver: local
```

**Configuration:**
```bash
MINIO_ENDPOINT=minio:9000  # Docker network hostname
MINIO_USE_SSL=true         # Enable SSL in production
```

### Option 2: AWS S3 (Recommended for Cloud)

**No code changes required!** MinIO SDK is S3-compatible.

**Configuration:**
```bash
MINIO_ENDPOINT=s3.us-east-1.amazonaws.com
MINIO_ACCESS_KEY=${AWS_ACCESS_KEY_ID}
MINIO_SECRET_KEY=${AWS_SECRET_ACCESS_KEY}
MINIO_BUCKET=inflight-simulations-prod
MINIO_USE_SSL=true
```

**Bucket Policy (S3):**
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "AWS": "arn:aws:iam::ACCOUNT_ID:user/inflight-service"
      },
      "Action": [
        "s3:GetObject",
        "s3:PutObject",
        "s3:DeleteObject"
      ],
      "Resource": "arn:aws:s3:::inflight-simulations-prod/*"
    },
    {
      "Effect": "Allow",
      "Principal": {
        "AWS": "arn:aws:iam::ACCOUNT_ID:user/inflight-service"
      },
      "Action": [
        "s3:ListBucket"
      ],
      "Resource": "arn:aws:s3:::inflight-simulations-prod"
    }
  ]
}
```

---

## Monitoring

### Check Storage Usage

**MinIO Console:**
1. Login to http://localhost:9011
2. Navigate to Buckets → `inflight-simulations`
3. View "Usage" tab

**CLI:**
```bash
# Using MinIO Client (mc)
mc alias set local http://localhost:9010 admin admin_password
mc du local/inflight-simulations

# Expected output:
# 15.2 MiB   142 objects   local/inflight-simulations
```

### Database vs S3 Reconciliation

**Check for orphaned files:**
```sql
-- Files in database but not in S3
SELECT * FROM simulation_attachments
WHERE id NOT IN (
  SELECT id FROM simulation_attachments
  WHERE uploaded_at > NOW() - INTERVAL '1 hour'  -- Recently uploaded
);
```

**Cleanup orphaned S3 objects:**
```bash
# List all objects
mc ls --recursive local/inflight-simulations

# Compare with database and delete orphans
# (Requires custom script)
```

---

## Backup and Recovery

### Backup Strategy

**Option 1: MinIO Mirror**
```bash
# Mirror to another MinIO instance
mc mirror local/inflight-simulations remote/inflight-simulations-backup
```

**Option 2: Export to Filesystem**
```bash
# Download all files
mc cp --recursive local/inflight-simulations /backup/minio/inflight-simulations
```

**Option 3: S3 Replication** (MinIO Enterprise)
- Configure bucket replication to another MinIO instance
- Automatic synchronization

### Recovery

**Restore from backup:**
```bash
# Upload all files back to MinIO
mc cp --recursive /backup/minio/inflight-simulations local/inflight-simulations
```

**Note:** Database must also be restored for attachments to be accessible in UI.

---

## Security

### Access Control

**Current (Development):**
- Hardcoded credentials in `server.go`
- No encryption at rest
- No encryption in transit (HTTP)

**Recommended (Production):**
1. **Use environment variables:**
   ```bash
   export MINIO_ROOT_USER=strong-username
   export MINIO_ROOT_PASSWORD=strong-random-password
   ```

2. **Enable SSL:**
   ```bash
   MINIO_USE_SSL=true
   MINIO_ENDPOINT=minio.example.com:443
   ```

3. **Rotate credentials regularly**

4. **Use IAM policies** (MinIO supports AWS IAM-compatible policies)

5. **Enable encryption at rest:**
   ```bash
   # MinIO server with encryption
   mc admin config set local sse-s3 on
   ```

### Network Security

**Firewall Rules:**
- Block port 9010 from public internet
- Only allow UI service to connect
- Expose MinIO console (9011) via VPN or internal network only

**Example iptables:**
```bash
# Block MinIO API from public
iptables -A INPUT -p tcp --dport 9010 -s 0.0.0.0/0 -j DROP
iptables -A INPUT -p tcp --dport 9010 -s 10.0.0.0/8 -j ACCEPT  # Allow private network

# Block console from public
iptables -A INPUT -p tcp --dport 9011 -s 0.0.0.0/0 -j DROP
iptables -A INPUT -p tcp --dport 9011 -s 10.0.0.0/8 -j ACCEPT
```

---

## Troubleshooting

### MinIO Won't Start

**Check logs:**
```bash
docker logs metrics-cold-tier
```

**Common issues:**
- Port conflict (9010 or 9011 already in use)
- Volume mount permission denied
- Insufficient disk space

**Solutions:**
```bash
# Check port usage
netstat -tuln | grep 9010
netstat -tuln | grep 9011

# Fix volume permissions
sudo chown -R 1000:1000 /var/lib/docker/volumes/minio-data

# Check disk space
df -h
```

### Connection Refused

**Symptoms:** `Failed to initialize S3 attachment store: connection refused`

**Solutions:**
1. Verify MinIO is running:
   ```bash
   curl http://localhost:9010/minio/health/live
   ```

2. Check Docker network:
   ```bash
   docker network inspect metrics-network
   ```

3. Use correct endpoint:
   - Inside Docker: `minio:9000` (container name + internal port)
   - From host: `localhost:9010` (mapped port)

### Bucket Not Found

**Symptoms:** `The specified bucket does not exist`

**Solutions:**
1. Check bucket was created:
   ```bash
   mc ls local/
   ```

2. Create manually:
   ```bash
   mc mb local/inflight-simulations
   ```

3. Check service logs for bucket creation errors

---

## Migration from Filesystem

If you previously used filesystem storage (`/var/inflight/uploads`):

### 1. Export Existing Files

```bash
# Find all attachment records in database
psql -U simulator -d ui_service -c "SELECT id, storage_path FROM simulation_attachments;"

# Copy files to temporary location
mkdir /tmp/migration
cp -r /var/inflight/uploads/* /tmp/migration/
```

### 2. Upload to MinIO

```bash
# Upload all files maintaining directory structure
mc cp --recursive /tmp/migration/ local/inflight-simulations/simulations/

# Verify count matches
echo "Database count:"
psql -U simulator -d ui_service -c "SELECT COUNT(*) FROM simulation_attachments;"

echo "MinIO count:"
mc ls --recursive local/inflight-simulations | wc -l
```

### 3. Update Database Paths

```sql
-- Update storage_path to use S3 key format
UPDATE simulation_attachments
SET storage_path = 'simulations/' || storage_path
WHERE storage_path NOT LIKE 'simulations/%';
```

### 4. Verify

```bash
# Test download of a few attachments via API
curl -o test.png http://localhost:8080/api/v1/simulations/queue/142/attachments/1

# Check file integrity
file test.png
```

---

## Performance Tuning

### MinIO Optimization

**For high upload volume:**
```bash
# Increase API threads
MINIO_API_REQUESTS_MAX=1000

# Increase concurrent requests per node
MINIO_API_REQUESTS_DEADLINE=5m
```

**For large files:**
```bash
# Enable multipart upload threshold
# Files > 64MB use multipart automatically
```

### Network Optimization

**Use connection pooling:**
```go
// In server.go MinIO client initialization
transport := &http.Transport{
    MaxIdleConns:        100,
    MaxIdleConnsPerHost: 100,
    IdleConnTimeout:     90 * time.Second,
}

client, err := minio.New("localhost:9010", &minio.Options{
    Creds:     credentials.NewStaticV4("admin", "admin_password", ""),
    Secure:    false,
    Transport: transport,
})
```

---

## Cost Considerations

### Storage Costs

**MinIO (Self-Hosted):**
- Free software
- Cost = disk space + server costs
- Estimate: ~$0.05/GB/month (cloud VM storage)

**AWS S3 (Cloud):**
- Standard: $0.023/GB/month
- Infrequent Access: $0.0125/GB/month
- Glacier (archival): $0.004/GB/month

**Example:**
- 10,000 simulations
- Avg 2 attachments per simulation
- Avg 2 MB per attachment
- Total: 40 GB
- **MinIO cost:** ~$2/month
- **AWS S3 cost:** ~$1/month

### Transfer Costs

**MinIO:** Free (within same network)
**AWS S3:** $0.09/GB for data transfer out

---

## Backup Recommendations

### Development:
- **Frequency:** Weekly
- **Method:** `mc mirror` to external drive
- **Retention:** 2 weeks

### Production:
- **Frequency:** Daily
- **Method:** S3 cross-region replication OR `mc mirror` to DR site
- **Retention:** 90 days
- **Test restore:** Monthly

### Backup Script

```bash
#!/bin/bash
# backup-minio.sh

DATE=$(date +%Y%m%d)
BACKUP_DIR="/backup/minio/$DATE"

# Create backup directory
mkdir -p $BACKUP_DIR

# Mirror MinIO to backup location
mc mirror local/inflight-simulations $BACKUP_DIR/inflight-simulations

# Compress
tar -czf $BACKUP_DIR.tar.gz $BACKUP_DIR
rm -rf $BACKUP_DIR

# Cleanup old backups (keep last 30 days)
find /backup/minio -name "*.tar.gz" -mtime +30 -delete

echo "Backup complete: $BACKUP_DIR.tar.gz"
```

**Add to cron:**
```bash
# Daily at 2 AM
0 2 * * * /path/to/backup-minio.sh
```

---

## Monitoring Dashboard

### MinIO Metrics (Prometheus)

**Endpoint:** http://localhost:9010/minio/v2/metrics/cluster

**Key Metrics:**
- `minio_bucket_usage_total_bytes{bucket="inflight-simulations"}` - Storage used
- `minio_bucket_objects_count{bucket="inflight-simulations"}` - Object count
- `minio_s3_requests_total` - Request rate
- `minio_s3_errors_total` - Error rate

### Grafana Dashboard

**Import dashboard:** MinIO Dashboard (ID: 13502)

**Metrics to watch:**
- Storage usage trend
- Upload/download throughput
- Request latency (P99)
- Error rate

---

## Contact

For MinIO-related issues:
- **MinIO Docs:** https://min.io/docs/minio/linux/index.html
- **GitHub:** https://github.com/minio/minio
- **Community:** Slack (https://slack.min.io)

For Inflight-specific issues:
- Check `internal/storage/simulations/attachments_s3.go`
- Review logs: `docker logs inflight-ui-service`
- Test with `mc` command-line tool
