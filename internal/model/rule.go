package model

import (
	"database/sql"
	"time"
)

// RuleType 规则类型
type RuleType string

const (
	RuleTypeNamePrefix   RuleType = "name_prefix"   // 容器名称前缀匹配
	RuleTypeNameSuffix   RuleType = "name_suffix"   // 容器名称后缀匹配
	RuleTypeNameRegex    RuleType = "name_regex"    // 容器名称正则匹配
	RuleTypeNameContains RuleType = "name_contains" // 容器名称包含
	RuleTypeImagePrefix  RuleType = "image_prefix"  // 镜像名称前缀匹配
	RuleTypeImageRegex   RuleType = "image_regex"   // 镜像名称正则匹配
	RuleTypeLabelKey     RuleType = "label_key"     // 标签键存在
	RuleTypeLabelValue   RuleType = "label_value"   // 标签键值匹配 (格式: key=value)
)

// GroupRule 群组规则模型
type GroupRule struct {
	ID        int64     `json:"id"`
	GroupID   int64     `json:"groupId"`
	RuleType  RuleType  `json:"ruleType"`
	Pattern   string    `json:"pattern"`
	CreatedAt time.Time `json:"createdAt"`
}

// CreateRule 创建规则
func CreateRule(rule *GroupRule) (int64, error) {
	result, err := db.Exec(`
		INSERT INTO group_rules (group_id, rule_type, pattern)
		VALUES (?, ?, ?)
	`, rule.GroupID, rule.RuleType, rule.Pattern)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// DeleteRule 删除规则
func DeleteRule(id int64) error {
	_, err := db.Exec(`DELETE FROM group_rules WHERE id = ?`, id)
	return err
}

// DeleteRulesByGroupID 删除群组下的所有规则
func DeleteRulesByGroupID(groupID int64) error {
	_, err := db.Exec(`DELETE FROM group_rules WHERE group_id = ?`, groupID)
	return err
}

// GetRuleByID 根据ID获取规则
func GetRuleByID(id int64) (*GroupRule, error) {
	row := db.QueryRow(`
		SELECT id, group_id, rule_type, pattern, created_at
		FROM group_rules WHERE id = ?
	`, id)

	return scanRule(row)
}

// GetRulesByGroupID 获取群组下的所有规则
func GetRulesByGroupID(groupID int64) ([]GroupRule, error) {
	rows, err := db.Query(`
		SELECT id, group_id, rule_type, pattern, created_at
		FROM group_rules WHERE group_id = ? ORDER BY id ASC
	`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []GroupRule
	for rows.Next() {
		rule, err := scanRuleFromRows(rows)
		if err != nil {
			return nil, err
		}
		rules = append(rules, *rule)
	}

	return rules, rows.Err()
}

// GetAllRules 获取所有规则
func GetAllRules() ([]GroupRule, error) {
	rows, err := db.Query(`
		SELECT id, group_id, rule_type, pattern, created_at
		FROM group_rules ORDER BY group_id ASC, id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []GroupRule
	for rows.Next() {
		rule, err := scanRuleFromRows(rows)
		if err != nil {
			return nil, err
		}
		rules = append(rules, *rule)
	}

	return rules, rows.Err()
}

// scanRule 从单行扫描规则
func scanRule(row *sql.Row) (*GroupRule, error) {
	var rule GroupRule
	err := row.Scan(&rule.ID, &rule.GroupID, &rule.RuleType, &rule.Pattern, &rule.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

// scanRuleFromRows 从多行扫描规则
func scanRuleFromRows(rows *sql.Rows) (*GroupRule, error) {
	var rule GroupRule
	err := rows.Scan(&rule.ID, &rule.GroupID, &rule.RuleType, &rule.Pattern, &rule.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &rule, nil
}
