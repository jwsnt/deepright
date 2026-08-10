package knowledge

import (
	"fmt"
	"time"

	"knowledge/knowledgecore"
)

// LoadLastUpdateText loads the shared knowledge last update timestamp from the
// integration app directory and formats it for HTTP / CLI output.
func LoadLastUpdateText(appDir string, agentID string, loc *time.Location) (string, error) {
	db, err := knowledgecore.OpenSharedDB(appDir)
	if err != nil {
		return "", fmt.Errorf("open knowledge runtime: %w", err)
	}
	lastUpdate, err := knowledgecore.GetLastUpdateForAgent(db, agentID)
	if err != nil {
		return "", fmt.Errorf("load knowledge last update: %w", err)
	}
	if loc == nil {
		loc = time.Local
	}
	return time.UnixMilli(lastUpdate).In(loc).Format("2006-01-02 15:04"), nil
}
