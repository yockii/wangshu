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

func ClearChannels() {
	channelMu.Lock()
	defer channelMu.Unlock()
	for _, ch := range Channels {
		ch.Stop()
	}
	Channels = make(map[string]Channel)
}

func ClearChannelsExcept(excludeNames []string) {
	channelMu.Lock()
	defer channelMu.Unlock()

	excludeMap := make(map[string]bool)
	for _, name := range excludeNames {
		excludeMap[name] = true
	}

	for name, ch := range Channels {
		if !excludeMap[name] {
			ch.Stop()
			delete(Channels, name)
		}
	}
}
