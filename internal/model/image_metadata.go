package model

import (
	"database/sql"
	"time"
)

// 镜像来源类型
const (
	SourceTypeRemote  = "remote"  // 远程镜像（Docker Hub 或公共 Registry）
	SourceTypeLocal   = "local"   // 本地镜像（不检查更新）
	SourceTypePrivate = "private" // 私有 Registry 镜像
)

// ImageMetadata 镜像元数据
type ImageMetadata struct {
	ID             int64      `json:"id"`
	ImageID        string     `json:"imageId"`
	ImageName      string     `json:"imageName"`
	ImageTag       string     `json:"imageTag"`
	SourceType     string     `json:"sourceType"`
	RegistryHost   string     `json:"registryHost,omitempty"`
	LastCheckAt    *time.Time `json:"lastCheckAt,omitempty"`
	LastCheckError string     `json:"lastCheckError,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

// GetImageMetadata 根据 ImageID 获取镜像元数据
func GetImageMetadata(imageID string) (*ImageMetadata, error) {
	row := db.QueryRow(`
		SELECT id, image_id, image_name, image_tag, source_type, registry_host,
		       last_check_at, last_check_error, created_at, updated_at
		FROM image_metadata WHERE image_id = ?
	`, imageID)

	return scanImageMetadata(row)
}

// GetImageMetadataByName 根据镜像名和标签获取元数据
func GetImageMetadataByName(imageName, imageTag string) (*ImageMetadata, error) {
	row := db.QueryRow(`
		SELECT id, image_id, image_name, image_tag, source_type, registry_host,
		       last_check_at, last_check_error, created_at, updated_at
		FROM image_metadata WHERE image_name = ? AND image_tag = ?
	`, imageName, imageTag)

	return scanImageMetadata(row)
}

// UpsertImageMetadata 创建或更新镜像元数据
func UpsertImageMetadata(meta *ImageMetadata) error {
	now := time.Now()
	_, err := db.Exec(`
		INSERT INTO image_metadata (image_id, image_name, image_tag, source_type, registry_host, last_check_at, last_check_error, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(image_id) DO UPDATE SET
			image_name = excluded.image_name,
			image_tag = excluded.image_tag,
			source_type = excluded.source_type,
			registry_host = excluded.registry_host,
			last_check_at = excluded.last_check_at,
			last_check_error = excluded.last_check_error,
			updated_at = excluded.updated_at
	`, meta.ImageID, meta.ImageName, meta.ImageTag, meta.SourceType, meta.RegistryHost, meta.LastCheckAt, meta.LastCheckError, now)
	return err
}

// UpdateImageSourceType 更新镜像来源类型
func UpdateImageSourceType(imageID string, sourceType string, checkError string) error {
	now := time.Now()
	_, err := db.Exec(`
		UPDATE image_metadata
		SET source_type = ?, last_check_at = ?, last_check_error = ?, updated_at = ?
		WHERE image_id = ?
	`, sourceType, now, checkError, now, imageID)
	return err
}

// GetAllImageMetadata 获取所有镜像元数据
func GetAllImageMetadata() ([]ImageMetadata, error) {
	rows, err := db.Query(`
		SELECT id, image_id, image_name, image_tag, source_type, registry_host,
		       last_check_at, last_check_error, created_at, updated_at
		FROM image_metadata ORDER BY updated_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []ImageMetadata
	for rows.Next() {
		var meta ImageMetadata
		var registryHost sql.NullString
		var lastCheckAt sql.NullTime
		var lastCheckError sql.NullString

		err := rows.Scan(
			&meta.ID, &meta.ImageID, &meta.ImageName, &meta.ImageTag,
			&meta.SourceType, &registryHost, &lastCheckAt, &lastCheckError,
			&meta.CreatedAt, &meta.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		meta.RegistryHost = registryHost.String
		if lastCheckAt.Valid {
			meta.LastCheckAt = &lastCheckAt.Time
		}
		meta.LastCheckError = lastCheckError.String

		results = append(results, meta)
	}

	return results, rows.Err()
}

// GetImageMetadataMap 获取镜像元数据的 Map（按 ImageID 索引）
func GetImageMetadataMap() (map[string]*ImageMetadata, error) {
	metas, err := GetAllImageMetadata()
	if err != nil {
		return nil, err
	}

	result := make(map[string]*ImageMetadata)
	for i := range metas {
		result[metas[i].ImageID] = &metas[i]
	}
	return result, nil
}

// DeleteImageMetadata 删除镜像元数据
func DeleteImageMetadata(imageID string) error {
	_, err := db.Exec(`DELETE FROM image_metadata WHERE image_id = ?`, imageID)
	return err
}

// scanImageMetadata 从单行扫描镜像元数据
func scanImageMetadata(row *sql.Row) (*ImageMetadata, error) {
	var meta ImageMetadata
	var registryHost sql.NullString
	var lastCheckAt sql.NullTime
	var lastCheckError sql.NullString

	err := row.Scan(
		&meta.ID, &meta.ImageID, &meta.ImageName, &meta.ImageTag,
		&meta.SourceType, &registryHost, &lastCheckAt, &lastCheckError,
		&meta.CreatedAt, &meta.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	meta.RegistryHost = registryHost.String
	if lastCheckAt.Valid {
		meta.LastCheckAt = &lastCheckAt.Time
	}
	meta.LastCheckError = lastCheckError.String

	return &meta, nil
}
