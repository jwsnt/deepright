package main

import (
	"connect/connectsvc"
)

var localPluginDirResolver = connectsvc.DefaultLocalPluginDir

func listLocalPluginMeta(svc *connectsvc.Service) ([]connectsvc.PluginMetaInfo, error) {
	return connectsvc.ListLocalPluginMetaWithService(svc, connectsvc.LocalPluginOptions{
		ResolveDir: localPluginDirResolver,
	})
}

func listLocalPlugins() ([]connectsvc.PluginInfo, error) {
	return connectsvc.ListLocalPlugins(connectsvc.LocalPluginOptions{
		ResolveDir: localPluginDirResolver,
	})
}
