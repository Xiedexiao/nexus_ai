package service

import (
	"regexp"
	"sync"
	"sync/atomic"

	"github.com/QuantumNous/new-api/setting/model_setting"
)

// Chat→Responses upgrade policy is host routing logic (it decides *whether*
// to convert, reading host settings), so it lives here, not in relayconvert.

// maxCachedRegexes bounds every admin-pattern regex cache. Patterns come from
// admin settings (never request input), so the working set is small; the cap
// only stops a misconfigured rule from growing the map without bound. Past the
// cap new patterns still compile and match, they are simply not retained.
const maxCachedRegexes = 512

// regexCache is a compile-once cache for admin-configured patterns with a
// loose size bound: once full, entries stop being admitted until the process
// restarts or the helper is reset by tests. Admin patterns are few and stable,
// so the bound is never hit in practice.
type regexCache struct {
	entries sync.Map // pattern -> *regexp.Regexp
	size    atomic.Int64
}

func (c *regexCache) get(pattern string) (*regexp.Regexp, bool) {
	v, ok := c.entries.Load(pattern)
	if !ok {
		return nil, false
	}
	re, ok := v.(*regexp.Regexp)
	return re, ok
}

func (c *regexCache) store(pattern string, re *regexp.Regexp) {
	if c.size.Add(1) > maxCachedRegexes {
		// Keep memory bounded; recompiling on later requests is correct, just
		// slower, and only happens while an admin floods distinct patterns.
		c.size.Add(-1)
		return
	}
	c.entries.Store(pattern, re)
}

var chatResponsesRegexCache regexCache

func matchAnyModelPattern(patterns []string, model string) bool {
	if len(patterns) == 0 || model == "" {
		return false
	}
	for _, pattern := range patterns {
		if pattern == "" {
			continue
		}
		re, ok := chatResponsesRegexCache.get(pattern)
		if !ok {
			compiled, err := regexp.Compile(pattern)
			if err != nil {
				// Treat invalid patterns as non-matching to avoid breaking runtime traffic.
				continue
			}
			re = compiled
			chatResponsesRegexCache.store(pattern, re)
		}
		if re.MatchString(model) {
			return true
		}
	}
	return false
}

func ShouldChatCompletionsUseResponsesPolicy(policy model_setting.ChatCompletionsToResponsesPolicy, channelID int, channelType int, model string) bool {
	if !policy.IsChannelEnabled(channelID, channelType) {
		return false
	}
	return matchAnyModelPattern(policy.ModelPatterns, model)
}

func ShouldChatCompletionsUseResponsesGlobal(channelID int, channelType int, model string) bool {
	return ShouldChatCompletionsUseResponsesPolicy(
		model_setting.GetGlobalSettings().ChatCompletionsToResponsesPolicy,
		channelID,
		channelType,
		model,
	)
}
