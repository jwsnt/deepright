package connectsvc

import (
	"errors"
	"reflect"
	"testing"
)

func TestBuildRuntimeSkillNames(t *testing.T) {
	originalLookup := internalSkillStatusLookup
	internalSkillStatusLookup = func(key string, _ map[string]string) (*PluginStatus, error) {
		switch key {
		case "browser":
			return &PluginStatus{Key: key, Started: true}, nil
		case "email", "feishu":
			return &PluginStatus{Key: key, Started: true}, nil
		default:
			return nil, errors.New("missing")
		}
	}
	defer func() {
		internalSkillStatusLookup = originalLookup
	}()

	got := BuildRuntimeSkillNames([]string{"alpha", InternalSkillCron, "beta"})
	want := []string{"alpha", InternalSkillCron, "beta", InternalSkillBrowser, InternalSkillEmail, InternalSkillFeishu}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("BuildRuntimeSkillNames() = %#v, want %#v", got, want)
	}
}
