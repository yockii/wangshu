package bus

var defaultBus = NewMessageBus(100)

func Default() *MessageBus {
	return defaultBus
}

func Close() {
	defaultBus.Close()
}
