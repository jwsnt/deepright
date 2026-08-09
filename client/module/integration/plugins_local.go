package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"connect/connectsvc"
)

var localPluginDirResolver = defaultLocalPluginDir

func defaultLocalPluginDir() (string, error) {
	if explicit := strings.TrimSpace(os.Getenv(integrationPluginDirEnv)); explicit != "" {
		return filepath.Clean(explicit), nil
	}
	if runtime.GOOS == "darwin" {
		if pluginDir := strings.TrimSpace(integrationBundledPluginDir()); pluginDir != "" {
			return filepath.Clean(pluginDir), nil
		}
		return "", fmt.Errorf("resolve application plugins dir")
	}
	if explicit := strings.TrimSpace(os.Getenv(integrationPluginDirEnv)); explicit != "" {
		return filepath.Clean(explicit), nil
	}
	return connectsvc.DefaultLocalPluginDir()
}

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
