package converter

import (
	"reflect"
	"strings"
	"sync"

	"github.com/pingidentity/pingone-go-client/pingone"
)

var (
	allowedFlowSettingsOnce sync.Once
	allowedFlowSettingsKeys map[string]struct{}
)

// getAllowedFlowSettingsKeys lazily builds the set of flow setting keys supported by the PingOne client.
func getAllowedFlowSettingsKeys() map[string]struct{} {
	allowedFlowSettingsOnce.Do(func() {
		allowedFlowSettingsKeys = extractFlowSettingKeys()
	})
	return allowedFlowSettingsKeys
}

// extractFlowSettingKeys enumerates JSON tag names from the PingOne client request model to avoid hardcoding keys.
func extractFlowSettingKeys() map[string]struct{} {
	return collectJSONTaggedFields(reflect.TypeOf(pingone.DaVinciFlowSettingsRequest{}))
}

func collectJSONTaggedFields(t reflect.Type) map[string]struct{} {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	keys := make(map[string]struct{})
	if t.Kind() != reflect.Struct {
		return keys
	}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.Anonymous {
			nested := collectJSONTaggedFields(field.Type)
			for name := range nested {
				keys[name] = struct{}{}
			}
			continue
		}
		if field.PkgPath != "" { // unexported
			continue
		}
		tag := field.Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}
		name := tag
		if idx := strings.Index(tag, ","); idx >= 0 {
			name = tag[:idx]
		}
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		keys[name] = struct{}{}
	}
	return keys
}

// filterFlowSettings returns only the subset of settings supported by the PingOne models.
func filterFlowSettings(settings map[string]interface{}) map[string]interface{} {
	if len(settings) == 0 {
		return nil
	}
	allowed := getAllowedFlowSettingsKeys()
	filtered := make(map[string]interface{}, len(settings))
	for key, value := range settings {
		if _, ok := allowed[key]; !ok {
			continue
		}
		filtered[key] = value
	}
	if len(filtered) == 0 {
		return nil
	}
	return filtered
}
