package core

import "errors"

// RuleLifeCycle 检测规则生命周期
type RuleLifeCycle int

const (
	// RuleLifeCycleRuntime 运行时检测规则
	RuleLifeCycleRuntime RuleLifeCycle = 1

	// RuleLifeCycleStatic 静态检测规则
	RuleLifeCycleStatic RuleLifeCycle = 2
)

// Validate 验证表达式格式是否正确
func (e *RuleExpression) Validate() error {
	if e.Lang == "" || e.Expr == "" {
		return errors.New("表达式语言和表达式内容不能为空")
	}
	return nil
}

// RuleExpression 规则表达式,用于定义检测逻辑
type RuleExpression struct {
	// Lang 表达式语言类型,例如：rego、cel、sql 等
	Lang string `json:"lang"`
	// Expr 具体的表达式内容,根据 Lang 字段指定的语言编写
	Expr string `json:"expr"`
}

// NewAssetRule 创建新的资产检测规则
func NewAssetRule(code, name string, lifeCycle RuleLifeCycle, desc string, expression RuleExpression) (*AssetFinderRule, error) {
	if err := expression.Validate(); err != nil {
		return nil, err
	}
	return &AssetFinderRule{
		Code:       code,
		Name:       name,
		LifeCycle:  lifeCycle,
		Desc:       desc,
		Expression: expression,
	}, nil
}

// AssetFinderRule 资产检测规则定义
type AssetFinderRule struct {
	// Code 规则唯一标识,用于区分不同规则
	Code string `json:"code"`
	// Name 规则名称,便于用户理解规则用途
	Name string `json:"name"`
	// LifeCycle 定义规则的生命周期（运行时/静态）
	LifeCycle RuleLifeCycle `json:"life_cycle"`

	// OS 定义规则适用的操作系统列表（如 ["darwin", "linux", "windows"]）
	// 如果为空,表示适用于所有操作系统
	OS []string `json:"os,omitempty"`

	// Desc 规则描述,说明规则具体作用
	Desc string `json:"desc"`

	// Expression 定义检测逻辑表达式
	Expression RuleExpression `json:"expression"`
}

// Validate 验证资产规则是否正确
func (r *AssetFinderRule) Validate() error {
	if r.Code == "" || r.Name == "" || r.Desc == "" {
		return errors.New("资产规则代码、名称、描述不能为空")
	}
	if r.LifeCycle != RuleLifeCycleRuntime && r.LifeCycle != RuleLifeCycleStatic {
		return errors.New("资产规则生命周期必须是运行时或静态检测规则")
	}
	if err := r.Expression.Validate(); err != nil {
		return err
	}
	return nil
}
