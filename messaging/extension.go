package messaging

import "strings"

type ExtensionKey string

const (
	SysExtensionKeyPrefix             string       = "squat:"
	ExtensionKeyNoticeServiceEndpoint ExtensionKey = ExtensionKey(SysExtensionKeyPrefix) + "NoticeServiceEndpoint"
	ExtensionKeyFromMessageID         ExtensionKey = ExtensionKey(SysExtensionKeyPrefix) + "FromMessageID"
	ExtensionKeyFromMessageType       ExtensionKey = ExtensionKey(SysExtensionKeyPrefix) + "FromMessageType"
	ExtensionKeyAggregateChanged      ExtensionKey = ExtensionKey(SysExtensionKeyPrefix) + "AggregateChanged"
)

// Extensions is not concurrent-safe.
type Extensions map[string]string

func (extensions Extensions) Clone() Extensions {
	if len(extensions) == 0 {
		return nil
	}
	cloned := make(map[string]string)
	for key, val := range extensions {
		cloned[key] = val
	}
	return cloned
}

func (extensions Extensions) CustomExtensions() Extensions {
	if len(extensions) == 0 {
		return nil
	}
	customExtensions := make(map[string]string)
	for key, val := range extensions {
		if !strings.HasPrefix(key, SysExtensionKeyPrefix) {
			customExtensions[key] = val
		}
	}
	return customExtensions
}

func (extensions Extensions) Get(key ExtensionKey) (val string, ok bool) {
	if len(extensions) == 0 {
		return "", false
	}
	val, ok = extensions[string(key)]
	return
}

func (extensions Extensions) Set(key ExtensionKey, val string) Extensions {
	if extensions == nil {
		extensions = make(Extensions)
	}
	extensions[string(key)] = val
	return extensions
}

func (extensions Extensions) Remove(keys ...ExtensionKey) Extensions {
	if len(extensions) > 0 && len(keys) > 0 {
		for _, key := range keys {
			delete(extensions, string(key))
		}
	}
	return extensions
}

func (extensions Extensions) Merge(toMerge Extensions) Extensions {
	if len(extensions) == 0 {
		return toMerge
	}
	for key, val := range toMerge {
		extensions[key] = val
	}
	return extensions
}
