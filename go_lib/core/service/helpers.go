package service

// ========== 响应格式化辅助函数 ==========

// successResult 成功响应（无数据）
func successResult() map[string]interface{} {
	return map[string]interface{}{
		"success": true,
	}
}

// successDataResult 成功响应（带数据）
func successDataResult(data interface{}) map[string]interface{} {
	return map[string]interface{}{
		"success": true,
		"data":    data,
	}
}

// errorResult 错误响应
func errorResult(err error) map[string]interface{} {
	return map[string]interface{}{
		"success": false,
		"error":   err.Error(),
	}
}

// errorMessageResult 错误响应（直接传消息）
func errorMessageResult(msg string) map[string]interface{} {
	return map[string]interface{}{
		"success": false,
		"error":   msg,
	}
}
