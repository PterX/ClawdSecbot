package callback_bridge

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go_lib/core/logging"
)

// MessageType 消息类型
type MessageType string

const (
	MessageTypeLog           MessageType = "log"
	MessageTypeMetrics       MessageType = "metrics"
	MessageTypeStatus        MessageType = "status"
	MessageTypeVersionUpdate MessageType = "version_update"
	MessageTypeSecurityEvent MessageType = "security_event"
)

// Message 统一消息结构
type Message struct {
	Type      MessageType            `json:"type"`
	Timestamp int64                  `json:"timestamp"`
	Payload   map[string]interface{} `json:"payload"`
}

// CallbackFunc Go 回调函数类型
type CallbackFunc func(message string)

// Bridge 回调桥接器,用于 Go 和 Dart 之间的通信
type Bridge struct {
	// Go 回调函数（由外部设置）
	callback CallbackFunc

	// 输入通道
	logChan           chan string
	metricsChan       chan map[string]interface{}
	securityEventChan chan map[string]interface{}

	// 生命周期管理
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// 状态
	running bool
	mu      sync.Mutex
}

// NewBridge 创建回调桥接器
func NewBridge(callback CallbackFunc) (*Bridge, error) {
	if callback == nil {
		return nil, fmt.Errorf("callback function is required")
	}

	ctx, cancel := context.WithCancel(context.Background())

	bridge := &Bridge{
		callback:          callback,
		logChan:           make(chan string, 1000),
		metricsChan:       make(chan map[string]interface{}, 100),
		securityEventChan: make(chan map[string]interface{}, 100),
		ctx:               ctx,
		cancel:            cancel,
		running:           true,
	}

	// 启动发布工作协程
	bridge.wg.Add(1)
	go bridge.publishWorker()

	logging.Info("[CallbackBridge] Callback bridge initialized")
	return bridge, nil
}

// publishWorker 发布日志、指标和安全事件的工作协程
func (b *Bridge) publishWorker() {
	defer b.wg.Done()

	// 批量发送定时器
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	var logBatch []string
	var metricsBatch []map[string]interface{}

	for {
		select {
		case <-b.ctx.Done():
			// 发送剩余消息
			b.flushLogs(logBatch)
			b.flushMetrics(metricsBatch)
			b.drainSecurityEvents()
			return

		case log := <-b.logChan:
			logBatch = append(logBatch, log)
			if len(logBatch) >= 5 {
				b.flushLogs(logBatch)
				logBatch = logBatch[:0]
			}

		case metrics := <-b.metricsChan:
			metricsBatch = append(metricsBatch, metrics)
			if len(metricsBatch) >= 5 {
				b.flushMetrics(metricsBatch)
				metricsBatch = metricsBatch[:0]
			}

		case event := <-b.securityEventChan:
			// 安全事件频率低、实时性要求高，收到即 flush，不做批量合并
			b.flushSecurityEvents([]map[string]interface{}{event})

		case <-ticker.C:
			// 定时刷新
			if len(logBatch) > 0 {
				b.flushLogs(logBatch)
				logBatch = logBatch[:0]
			}
			if len(metricsBatch) > 0 {
				b.flushMetrics(metricsBatch)
				metricsBatch = metricsBatch[:0]
			}
		}
	}
}

// invokeCallback 调用回调发送消息
func (b *Bridge) invokeCallback(data []byte) error {
	b.mu.Lock()
	callback := b.callback
	running := b.running
	b.mu.Unlock()

	if !running || callback == nil {
		return fmt.Errorf("bridge not running")
	}

	if len(data) == 0 {
		return fmt.Errorf("empty data")
	}

	if data[0] != '{' || data[len(data)-1] != '}' {
		return fmt.Errorf("invalid JSON format")
	}

	defer func() {
		if r := recover(); r != nil {
			logging.Error("[CallbackBridge] Recovered from callback panic: %v", r)
		}
	}()

	callback(string(data))

	return nil
}

// flushLogs 发送日志批次
func (b *Bridge) flushLogs(logs []string) {
	for _, log := range logs {
		msg := b.newLogMessage(log)
		if data, err := json.Marshal(msg); err == nil {
			_ = b.invokeCallback(data)
		}
	}
}

// flushMetrics 发送指标批次
func (b *Bridge) flushMetrics(metricsList []map[string]interface{}) {
	for _, metrics := range metricsList {
		msg := b.newMetricsMessage(metrics)
		if data, err := json.Marshal(msg); err == nil {
			_ = b.invokeCallback(data)
		}
	}
}

// flushSecurityEvents 发送安全事件批次
func (b *Bridge) flushSecurityEvents(events []map[string]interface{}) {
	for _, event := range events {
		msg := &Message{
			Type:      MessageTypeSecurityEvent,
			Timestamp: time.Now().UnixMilli(),
			Payload:   event,
		}
		if data, err := json.Marshal(msg); err == nil {
			_ = b.invokeCallback(data)
		}
	}
}

// drainSecurityEvents 关闭时清空残留安全事件
func (b *Bridge) drainSecurityEvents() {
	for {
		select {
		case event := <-b.securityEventChan:
			b.flushSecurityEvents([]map[string]interface{}{event})
		default:
			return
		}
	}
}

// newLogMessage 从已有的 JSON 字符串创建日志消息
func (b *Bridge) newLogMessage(jsonStr string) *Message {
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &payload); err != nil {
		payload = map[string]interface{}{"message": jsonStr}
	}
	return &Message{
		Type:      MessageTypeLog,
		Timestamp: time.Now().UnixMilli(),
		Payload:   payload,
	}
}

// newMetricsMessage 创建性能指标消息
func (b *Bridge) newMetricsMessage(metrics map[string]interface{}) *Message {
	return &Message{
		Type:      MessageTypeMetrics,
		Timestamp: time.Now().UnixMilli(),
		Payload:   metrics,
	}
}

// newStatusMessage 创建状态消息
func (b *Bridge) newStatusMessage(status map[string]interface{}) *Message {
	return &Message{
		Type:      MessageTypeStatus,
		Timestamp: time.Now().UnixMilli(),
		Payload:   status,
	}
}

// SendLog 发送日志 (非阻塞)
func (b *Bridge) SendLog(log string) {
	select {
	case b.logChan <- log:
	default:
		// 通道满,丢弃最旧的消息
		select {
		case <-b.logChan:
		default:
		}
		select {
		case b.logChan <- log:
		default:
		}
	}
}

// SendMetrics 发送指标 (非阻塞)
func (b *Bridge) SendMetrics(metrics map[string]interface{}) {
	select {
	case b.metricsChan <- metrics:
	default:
		// 通道满时丢弃最旧项，尽量保留最新快照，减少统计落后/漏同步窗口。
		select {
		case <-b.metricsChan:
		default:
		}
		select {
		case b.metricsChan <- metrics:
		default:
		}
	}
}

// SendStatus 发送状态消息 (直接发送,不走批量)
func (b *Bridge) SendStatus(status map[string]interface{}) {
	msg := b.newStatusMessage(status)
	if data, err := json.Marshal(msg); err == nil {
		_ = b.invokeCallback(data)
	}
}

// SendVersionUpdate 发送版本更新消息 (直接发送,不走批量)
func (b *Bridge) SendVersionUpdate(versionInfo map[string]interface{}) {
	msg := &Message{
		Type:      MessageTypeVersionUpdate,
		Timestamp: time.Now().UnixMilli(),
		Payload:   versionInfo,
	}
	if data, err := json.Marshal(msg); err == nil {
		_ = b.invokeCallback(data)
	}
}

// SendSecurityEvent 发送安全事件消息 (通过 channel 走 publishWorker 序列化路径)
func (b *Bridge) SendSecurityEvent(event map[string]interface{}) {
	select {
	case b.securityEventChan <- event:
	default:
		// 通道满,跳过
	}
}

// IsRunning 返回是否正在运行
func (b *Bridge) IsRunning() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.running
}

// Close 关闭桥接器
func (b *Bridge) Close() error {
	b.mu.Lock()
	if !b.running {
		b.mu.Unlock()
		return nil
	}
	b.running = false
	b.mu.Unlock()

	b.cancel()
	b.wg.Wait()

	logging.Info("[CallbackBridge] Callback bridge closed")
	return nil
}
