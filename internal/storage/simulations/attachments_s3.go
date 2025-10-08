package simulations

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// AttachmentType represents the category of attachment
type AttachmentType string

const (
	AttachmentTypeScreenshot     AttachmentType = "screenshot"
	AttachmentTypeConfig         AttachmentType = "config"
	AttachmentTypeLog            AttachmentType = "log"
	AttachmentTypeDocumentation  AttachmentType = "documentation"
	AttachmentTypeOther          AttachmentType = "other"
)

// SimulationAttachment represents a file attached to a simulation job
type SimulationAttachment struct {
	ID              int            `db:"id" json:"id"`
	SimulationJobID int            `db:"simulation_job_id" json:"simulation_job_id"`
	UserID          int            `db:"user_id" json:"user_id"`
	FileName        string         `db:"file_name" json:"file_name"`
	FileType        string         `db:"file_type" json:"file_type"` // MIME type
	FileSize        int            `db:"file_size" json:"file_size"` // bytes
	StoragePath     string         `db:"storage_path" json:"storage_path"`
	AttachmentType  AttachmentType `db:"attachment_type" json:"attachment_type"`
	Description     *string        `db:"description" json:"description,omitempty"`
	UploadedAt      time.Time      `db:"uploaded_at" json:"uploaded_at"`
}

// CreateAttachmentInput represents input for creating an attachment
type CreateAttachmentInput struct {
	SimulationJobID int
	UserID          int
	FileName        string
	FileType        string
	FileSize        int
	StoragePath     string
	AttachmentType  AttachmentType
	Description     *string
}

// S3AttachmentStore handles database operations and S3 storage for simulation attachments
type S3AttachmentStore struct {
	db           *sql.DB
	s3Client     *minio.Client
	bucketName   string
}

// NewS3AttachmentStore creates a new S3-backed attachment store
func NewS3AttachmentStore(db *sql.DB, endpoint, accessKey, secretKey, bucketName string, useSSL bool) (*S3AttachmentStore, error) {
	// Initialize MinIO client
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	store := &S3AttachmentStore{
		db:         db,
		s3Client:   minioClient,
		bucketName: bucketName,
	}

	// Ensure bucket exists
	if err := store.ensureBucket(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ensure bucket: %w", err)
	}

	return store, nil
}

// ensureBucket creates the bucket if it doesn't exist
func (s *S3AttachmentStore) ensureBucket(ctx context.Context) error {
	exists, err := s.s3Client.BucketExists(ctx, s.bucketName)
	if err != nil {
		return fmt.Errorf("check bucket exists: %w", err)
	}

	if !exists {
		if err := s.s3Client.MakeBucket(ctx, s.bucketName, minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("create bucket: %w", err)
		}
	}

	return nil
}

// CreateAttachment stores attachment metadata in database
func (s *S3AttachmentStore) CreateAttachment(ctx context.Context, input CreateAttachmentInput) (*SimulationAttachment, error) {
	query := `
		INSERT INTO simulation_attachments (
			simulation_job_id, user_id, file_name, file_type, file_size,
			storage_path, attachment_type, description
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, simulation_job_id, user_id, file_name, file_type, file_size,
		          storage_path, attachment_type, description, uploaded_at
	`

	var attachment SimulationAttachment
	err := s.db.QueryRowContext(ctx, query,
		input.SimulationJobID, input.UserID, input.FileName, input.FileType, input.FileSize,
		input.StoragePath, input.AttachmentType, input.Description,
	).Scan(
		&attachment.ID, &attachment.SimulationJobID, &attachment.UserID,
		&attachment.FileName, &attachment.FileType, &attachment.FileSize,
		&attachment.StoragePath, &attachment.AttachmentType, &attachment.Description,
		&attachment.UploadedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("create attachment: %w", err)
	}

	return &attachment, nil
}

// GetAttachment retrieves an attachment by ID
func (s *S3AttachmentStore) GetAttachment(ctx context.Context, attachmentID int) (*SimulationAttachment, error) {
	query := `
		SELECT id, simulation_job_id, user_id, file_name, file_type, file_size,
		       storage_path, attachment_type, description, uploaded_at
		FROM simulation_attachments
		WHERE id = $1
	`

	var attachment SimulationAttachment
	err := s.db.QueryRowContext(ctx, query, attachmentID).Scan(
		&attachment.ID, &attachment.SimulationJobID, &attachment.UserID,
		&attachment.FileName, &attachment.FileType, &attachment.FileSize,
		&attachment.StoragePath, &attachment.AttachmentType, &attachment.Description,
		&attachment.UploadedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get attachment: %w", err)
	}

	return &attachment, nil
}

// ListAttachments lists all attachments for a simulation job
func (s *S3AttachmentStore) ListAttachments(ctx context.Context, jobID int) ([]SimulationAttachment, error) {
	query := `
		SELECT id, simulation_job_id, user_id, file_name, file_type, file_size,
		       storage_path, attachment_type, description, uploaded_at
		FROM simulation_attachments
		WHERE simulation_job_id = $1
		ORDER BY uploaded_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, jobID)
	if err != nil {
		return nil, fmt.Errorf("list attachments: %w", err)
	}
	defer rows.Close()

	var attachments []SimulationAttachment
	for rows.Next() {
		var attachment SimulationAttachment
		err := rows.Scan(
			&attachment.ID, &attachment.SimulationJobID, &attachment.UserID,
			&attachment.FileName, &attachment.FileType, &attachment.FileSize,
			&attachment.StoragePath, &attachment.AttachmentType, &attachment.Description,
			&attachment.UploadedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan attachment: %w", err)
		}
		attachments = append(attachments, attachment)
	}

	return attachments, nil
}

// DeleteAttachment deletes an attachment (both metadata and S3 object)
func (s *S3AttachmentStore) DeleteAttachment(ctx context.Context, attachmentID int) error {
	// Get attachment to find S3 key
	attachment, err := s.GetAttachment(ctx, attachmentID)
	if err != nil {
		return fmt.Errorf("get attachment: %w", err)
	}
	if attachment == nil {
		return fmt.Errorf("attachment not found")
	}

	// Delete from database first
	query := `DELETE FROM simulation_attachments WHERE id = $1`
	result, err := s.db.ExecContext(ctx, query, attachmentID)
	if err != nil {
		return fmt.Errorf("delete attachment: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("attachment not found")
	}

	// Delete from S3 (best effort - don't fail if S3 delete fails)
	if err := s.s3Client.RemoveObject(ctx, s.bucketName, attachment.StoragePath, minio.RemoveObjectOptions{}); err != nil {
		// Log error but don't fail - database record is already deleted
		return fmt.Errorf("warning: failed to delete S3 object %s: %w", attachment.StoragePath, err)
	}

	return nil
}

// SaveFile uploads a file to MinIO S3
// Returns the S3 key (storage path) and file size
func (s *S3AttachmentStore) SaveFile(ctx context.Context, jobID int, userID int, fileName string, fileData io.Reader, contentType string, fileSize int64) (string, int64, error) {
	// Create S3 key: simulations/{jobID}/{userID}/{fileName}
	s3Key := fmt.Sprintf("simulations/%d/%d/%s", jobID, userID, fileName)

	// Upload to MinIO
	uploadInfo, err := s.s3Client.PutObject(ctx, s.bucketName, s3Key, fileData, fileSize, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", 0, fmt.Errorf("upload to S3: %w", err)
	}

	return s3Key, uploadInfo.Size, nil
}

// GetFileReader returns a reader for downloading a file from S3
func (s *S3AttachmentStore) GetFileReader(ctx context.Context, attachment *SimulationAttachment) (io.ReadCloser, error) {
	object, err := s.s3Client.GetObject(ctx, s.bucketName, attachment.StoragePath, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("get S3 object: %w", err)
	}

	return object, nil
}

// GetPresignedURL generates a temporary download URL (expires in 1 hour)
func (s *S3AttachmentStore) GetPresignedURL(ctx context.Context, attachment *SimulationAttachment) (string, error) {
	url, err := s.s3Client.PresignedGetObject(ctx, s.bucketName, attachment.StoragePath, time.Hour, nil)
	if err != nil {
		return "", fmt.Errorf("generate presigned URL: %w", err)
	}

	return url.String(), nil
}

// GetFilePath returns the S3 key (for compatibility with filesystem version)
func (s *S3AttachmentStore) GetFilePath(attachment *SimulationAttachment) string {
	return attachment.StoragePath
}

// EnsureUploadsDirectory is a no-op for S3 (bucket is created in constructor)
func (s *S3AttachmentStore) EnsureUploadsDirectory() error {
	return nil // Bucket already created
}

// GetTotalSizeForJob calculates total attachment size for a job
func (s *S3AttachmentStore) GetTotalSizeForJob(ctx context.Context, jobID int) (int64, error) {
	query := `
		SELECT COALESCE(SUM(file_size), 0)
		FROM simulation_attachments
		WHERE simulation_job_id = $1
	`

	var totalSize int64
	err := s.db.QueryRowContext(ctx, query, jobID).Scan(&totalSize)
	if err != nil {
		return 0, fmt.Errorf("get total size: %w", err)
	}

	return totalSize, nil
}

// HealthCheck checks if MinIO is reachable
func (s *S3AttachmentStore) HealthCheck(ctx context.Context) error {
	// List buckets as a simple health check
	_, err := s.s3Client.ListBuckets(ctx)
	if err != nil {
		return fmt.Errorf("MinIO health check failed: %w", err)
	}
	return nil
}
