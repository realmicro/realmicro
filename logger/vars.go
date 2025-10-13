package logger

const (
	defaultCallerKey    = "caller"
	defaultContentKey   = "content"
	defaultDurationKey  = "duration"
	defaultLevelKey     = "level"
	defaultSpanKey      = "span"
	defaultTimestampKey = "@timestamp"
	defaultTraceKey     = "trace"
	defaultTruncatedKey = "truncated"
)

var (
	callerKey    = defaultCallerKey
	contentKey   = defaultContentKey
	durationKey  = defaultDurationKey
	levelKey     = defaultLevelKey
	spanKey      = defaultSpanKey
	timestampKey = defaultTimestampKey
	traceKey     = defaultTraceKey
	truncatedKey = defaultTruncatedKey
)
