package channel

import "sync"

var channelMu sync.RWMutex
var Channels = make(map[string]Channel)

func RegisterChannel(name string, channel Channel) {
	channelMu.Lock()
	defer channelMu.Unlock()

	Channels[name] = channel

	channel.Start()
}

func GetChannel(name string) (Channel, bool) {
	channelMu.RLock()
	defer channelMu.RUnlock()

	channel, exists := Channels[name]
	return channel, exists
}

func StopAllChannel() {
	for _, ch := range Channels {
		ch.Stop()
	}
}
