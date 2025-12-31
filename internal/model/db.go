package model

import (
	"database/sql"
	"os"
	"path/filepath"
	"sync"

	_ "modernc.org/sqlite"
	"github.com/zeromicro/go-zero/core/logx"
)

var (
	db   *sql.DB
	once sync.Once
)

// InitDB 初始化数据库连接
func InitDB(dataDir string) (*sql.DB, error) {
	var initErr error
	once.Do(func() {
		// 确保数据目录存在
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			initErr = err
			return
		}

		dbPath := filepath.Join(dataDir, "dockercopilot.db")
		logx.Infof("初始化数据库: %s", dbPath)

		var err error
		db, err = sql.Open("sqlite", dbPath+"?_busy_timeout=5000&_journal_mode=WAL")
		if err != nil {
			initErr = err
			return
		}

		// 设置连接池参数
		db.SetMaxOpenConns(1) // SQLite 只支持单写入
		db.SetMaxIdleConns(1)

		// 执行数据库迁移
		if err := migrate(db); err != nil {
			initErr = err
			return
		}

		logx.Info("数据库初始化完成")
	})

	return db, initErr
}

// GetDB 获取数据库实例
func GetDB() *sql.DB {
	return db
}

// migrate 执行数据库迁移
func migrate(db *sql.DB) error {
	// 群组表
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS container_groups (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT UNIQUE NOT NULL,
			group_type TEXT NOT NULL DEFAULT 'container',
			cron_expr TEXT NOT NULL DEFAULT '',
			auto_update INTEGER NOT NULL DEFAULT 0,
			check_update INTEGER NOT NULL DEFAULT 1,
			priority INTEGER NOT NULL DEFAULT 100,
			enabled INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}

	// 添加 group_type 列（如果不存在，忽略错误）
	_, _ = db.Exec(`ALTER TABLE container_groups ADD COLUMN group_type TEXT NOT NULL DEFAULT 'container'`)

	// 添加容器操作字段（如果不存在，忽略错误）
	_, _ = db.Exec(`ALTER TABLE container_groups ADD COLUMN restart_after_update INTEGER NOT NULL DEFAULT 0`)
	_, _ = db.Exec(`ALTER TABLE container_groups ADD COLUMN start_containers INTEGER NOT NULL DEFAULT 0`)
	_, _ = db.Exec(`ALTER TABLE container_groups ADD COLUMN stop_containers INTEGER NOT NULL DEFAULT 0`)

	// 规则表
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS group_rules (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			group_id INTEGER NOT NULL,
			rule_type TEXT NOT NULL,
			pattern TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (group_id) REFERENCES container_groups(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}

	// 创建规则索引
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_group_rules_group_id ON group_rules(group_id)`)
	if err != nil {
		return err
	}

	// 手动分配容器表
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS group_containers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			group_id INTEGER NOT NULL,
			container_id TEXT NOT NULL,
			container_name TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (group_id) REFERENCES container_groups(id) ON DELETE CASCADE,
			UNIQUE(group_id, container_id)
		)
	`)
	if err != nil {
		return err
	}

	// 创建容器分配索引
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_group_containers_group_id ON group_containers(group_id)`)
	if err != nil {
		return err
	}
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_group_containers_container_id ON group_containers(container_id)`)
	if err != nil {
		return err
	}

	// 更新历史表
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS update_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			group_id INTEGER,
			container_id TEXT NOT NULL,
			container_name TEXT NOT NULL,
			old_image TEXT NOT NULL,
			new_image TEXT NOT NULL,
			status TEXT NOT NULL,
			message TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (group_id) REFERENCES container_groups(id) ON DELETE SET NULL
		)
	`)
	if err != nil {
		return err
	}

	// 创建历史索引
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_update_history_group_id ON update_history(group_id)`)
	if err != nil {
		return err
	}
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_update_history_created_at ON update_history(created_at)`)
	if err != nil {
		return err
	}

	// 系统设置表
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}

	// 任务表
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS tasks (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			message TEXT NOT NULL DEFAULT '',
			detail_msg TEXT NOT NULL DEFAULT '',
			percentage INTEGER NOT NULL DEFAULT 0,
			is_done INTEGER NOT NULL DEFAULT 0,
			task_type TEXT NOT NULL DEFAULT '',
			target_id TEXT NOT NULL DEFAULT '',
			target_name TEXT NOT NULL DEFAULT '',
			sub_tasks TEXT NOT NULL DEFAULT '[]',
			started_at DATETIME NOT NULL,
			finished_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}

	// 创建任务索引
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_tasks_is_done ON tasks(is_done)`)
	if err != nil {
		return err
	}
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_tasks_started_at ON tasks(started_at)`)
	if err != nil {
		return err
	}

	// 镜像元数据表
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS image_metadata (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			image_id TEXT NOT NULL UNIQUE,
			image_name TEXT NOT NULL,
			image_tag TEXT NOT NULL,
			source_type TEXT NOT NULL DEFAULT 'remote',
			registry_host TEXT,
			last_check_at DATETIME,
			last_check_error TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}
	_, _ = db.Exec(`CREATE INDEX IF NOT EXISTS idx_image_metadata_image_id ON image_metadata(image_id)`)
	_, _ = db.Exec(`CREATE INDEX IF NOT EXISTS idx_image_metadata_source_type ON image_metadata(source_type)`)

	// 环境表
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS environments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			description TEXT DEFAULT '',
			env_type TEXT NOT NULL DEFAULT 'local',
			url TEXT DEFAULT '',
			secret_key TEXT DEFAULT '',
			jwt_token TEXT DEFAULT '',
			token_expires_at DATETIME,
			is_default INTEGER NOT NULL DEFAULT 0,
			status TEXT DEFAULT 'unknown',
			last_check_at DATETIME,
			last_error TEXT DEFAULT '',
			container_count INTEGER DEFAULT 0,
			running_count INTEGER DEFAULT 0,
			stopped_count INTEGER DEFAULT 0,
			image_count INTEGER DEFAULT 0,
			volume_count INTEGER DEFAULT 0,
			cpu_cores INTEGER DEFAULT 0,
			memory_total INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}

	// 添加新字段（兼容旧数据库）
	_, _ = db.Exec(`ALTER TABLE environments ADD COLUMN volume_count INTEGER DEFAULT 0`)
	_, _ = db.Exec(`ALTER TABLE environments ADD COLUMN cpu_cores INTEGER DEFAULT 0`)
	_, _ = db.Exec(`ALTER TABLE environments ADD COLUMN memory_total INTEGER DEFAULT 0`)
	if err != nil {
		return err
	}

	// 创建环境索引
	_, _ = db.Exec(`CREATE INDEX IF NOT EXISTS idx_environments_env_type ON environments(env_type)`)
	_, _ = db.Exec(`CREATE INDEX IF NOT EXISTS idx_environments_is_default ON environments(is_default)`)
	_, _ = db.Exec(`CREATE INDEX IF NOT EXISTS idx_environments_status ON environments(status)`)

	return nil
}

// CloseDB 关闭数据库连接
func CloseDB() error {
	if db != nil {
		return db.Close()
	}
	return nil
}
