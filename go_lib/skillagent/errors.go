package skillagent

import (
	"errors"
	"fmt"
)

// 哨兵错误
var (
	// ErrSkillNotFound 技能未找到
	ErrSkillNotFound = errors.New("skill not found")

	// ErrSkillAlreadyExists 尝试注册已存在的技能
	ErrSkillAlreadyExists = errors.New("skill already exists")

	// ErrInvalidSkillFormat SKILL.md 格式无效
	ErrInvalidSkillFormat = errors.New("invalid skill format")

	// ErrMissingSkillMd SKILL.md 文件缺失
	ErrMissingSkillMd = errors.New("SKILL.md file not found")

	// ErrInvalidFrontmatter YAML 前置元数据无效
	ErrInvalidFrontmatter = errors.New("invalid YAML frontmatter")

	// ErrMissingRequiredField 必需字段缺失
	ErrMissingRequiredField = errors.New("missing required field")

	// ErrSkillNotActivated 技能未激活
	ErrSkillNotActivated = errors.New("skill not activated")

	// ErrNoSkillsDiscovered 未发现任何技能
	ErrNoSkillsDiscovered = errors.New("no skills discovered")

	// ErrModelNotSupported 模型不支持工具调用
	ErrModelNotSupported = errors.New("chat model does not support tool calling")

	// ErrExecutionTimeout 技能执行超时
	ErrExecutionTimeout = errors.New("skill execution timeout")

	// ErrExecutionFailed 技能执行失败
	ErrExecutionFailed = errors.New("skill execution failed")
)

// SkillError 表示与特定技能相关的错误
type SkillError struct {
	SkillName string
	Op        string
	Err       error
}

func (e *SkillError) Error() string {
	if e.SkillName != "" {
		return fmt.Sprintf("skill %q: %s: %v", e.SkillName, e.Op, e.Err)
	}
	return fmt.Sprintf("%s: %v", e.Op, e.Err)
}

func (e *SkillError) Unwrap() error {
	return e.Err
}

// NewSkillError 创建 SkillError
func NewSkillError(skillName, op string, err error) *SkillError {
	return &SkillError{
		SkillName: skillName,
		Op:        op,
		Err:       err,
	}
}

// ParseError 表示 SKILL.md 解析错误
type ParseError struct {
	Path    string
	Line    int
	Message string
	Err     error
}

func (e *ParseError) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("parse error at %s:%d: %s", e.Path, e.Line, e.Message)
	}
	return fmt.Sprintf("parse error at %s: %s", e.Path, e.Message)
}

func (e *ParseError) Unwrap() error {
	return e.Err
}

// NewParseError 创建 ParseError
func NewParseError(path string, line int, message string, err error) *ParseError {
	return &ParseError{
		Path:    path,
		Line:    line,
		Message: message,
		Err:     err,
	}
}

// ValidationError 表示校验错误
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error: field %q: %s", e.Field, e.Message)
}

// NewValidationError 创建 ValidationError
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}
