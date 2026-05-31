package auth

import (
	"encoding/json"
	"net/url"
)

func chooseAccount(items []json.RawMessage, currentAccountID string) (id, name string, candidates int) {
	if currentAccountID != "" {
		return currentAccountID, "", len(items)
	}
	if len(items) != 1 {
		return "", "", len(items)
	}
	var account struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	_ = json.Unmarshal(items[0], &account)
	return account.ID, account.Name, len(items)
}

func chooseZone(items []json.RawMessage, preferredZone string) (id, name string, zoneMap map[string]string) {
	zoneMap = map[string]string{}
	type zoneRecord struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	zones := []zoneRecord{}
	for _, item := range items {
		var zone zoneRecord
		if err := json.Unmarshal(item, &zone); err != nil {
			continue
		}
		if zone.ID != "" && zone.Name != "" {
			zoneMap[zone.Name] = zone.ID
			zones = append(zones, zone)
		}
	}
	if preferredZone != "" {
		if zoneID := zoneMap[preferredZone]; zoneID != "" {
			return zoneID, preferredZone, zoneMap
		}
	}
	if len(zones) == 1 {
		return zones[0].ID, zones[0].Name, zoneMap
	}
	return "", "", zoneMap
}

func accountListParams(accountID string) url.Values {
	params := url.Values{}
	if accountID != "" {
		params.Set("id", accountID)
	}
	return params
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
