package constant

const Default = "default"

const HEARTBEAT_OK = "HEARTBEAT_OK"

const (
	ReachCompressHistory = 200 // 触发历史压缩的message数量阈值
	KeptHistory          = 20  // 压缩后保留的message数量
	MaxHistoryChars      = 100000 // 触发历史压缩的总字符数阈值（100K字符）
	CompressDebounce     = 5 * 60 // 压缩防抖时间（秒），避免短时间内重复压缩
)
