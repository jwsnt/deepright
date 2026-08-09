package connectsvc

import "strings"

const (
	InternalSkillCron    = "__internal_cron"
	InternalSkillBrowser = "__internal_browser"
	InternalSkillEmail   = "__internal_email"
	InternalSkillFeishu  = "__internal_feishu"
)

var internalSkillStatusLookup = PluginStatusByKey

func BuildRuntimeSkillNames(base []string) []string {
	out := append([]string(nil), base...)
	seen := make(map[string]struct{}, len(base)+3)
	for _, name := range base {
		name = strings.TrimSpace(name)
		if name != "" {
			seen[name] = struct{}{}
		}
	}

	appendIfMissing := func(name string, enabled bool) {
		if !enabled {
			return
		}
		if _, ok := seen[name]; ok {
			return
		}
		out = append(out, name)
		seen[name] = struct{}{}
	}

	started := discoverStartedInternalPlugins()
	appendIfMissing(InternalSkillCron, true)
	appendIfMissing(InternalSkillBrowser, hasRuntimePlugin(started, "browser"))
	appendIfMissing(InternalSkillEmail, hasRuntimePlugin(started, "email"))
	appendIfMissing(InternalSkillFeishu, hasRuntimePlugin(started, "feishu"))
	return out
}

func discoverStartedInternalPlugins() map[string]struct{} {
	keys := make(map[string]struct{}, 3)
	for _, key := range []string{"browser", "email", "feishu"} {
		status, err := internalSkillStatusLookup(key, nil)
		if err != nil || status == nil || !status.Started {
			continue
		}
		keys[key] = struct{}{}
	}
	return keys
}

func hasRuntimePlugin(pluginKeys map[string]struct{}, key string) bool {
	if len(pluginKeys) == 0 {
		return false
	}
	_, ok := pluginKeys[strings.TrimSpace(key)]
	return ok
}
