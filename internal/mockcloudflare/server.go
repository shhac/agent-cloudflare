package mockcloudflare

import (
	"encoding/json"
	"net/http"
	"strings"
)

type Server struct {
	mux *http.ServeMux
}

func NewServer() http.Handler {
	s := &Server{mux: http.NewServeMux()}
	s.routes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("CF-Mock", "true")
	if r.URL.Path == "/" || r.URL.Path == "/healthz" {
		s.mux.ServeHTTP(w, r)
		return
	}
	if !hasBearerToken(r) {
		writeEnvelope(w, http.StatusUnauthorized, nil, []message{{Code: 9109, Message: "Valid user-level authentication not found"}}, nil)
		return
	}
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	s.mux.HandleFunc("/", s.handleIndex)
	s.mux.HandleFunc("/healthz", s.handleHealthz)
	s.mux.HandleFunc("/graphql", s.handleGraphQL)
	s.mux.HandleFunc("/user/tokens/verify", s.handleVerifyToken)
	s.mux.HandleFunc("/accounts", s.handleAccounts)
	s.mux.HandleFunc("/accounts/", s.handleAccountSubresources)
	s.mux.HandleFunc("/zones", s.handleZones)
	s.mux.HandleFunc("/zones/", s.handleZoneSubresources)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		writeEnvelope(w, http.StatusNotFound, nil, []message{{Code: 1003, Message: "route not found"}}, nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"service": "mockcloudflare",
		"routes":  Routes(),
	})
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleVerifyToken(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	writeEnvelope(w, http.StatusOK, map[string]any{
		"id":     "token_mock_123",
		"status": "active",
	}, nil, nil)
}

func (s *Server) handleAccounts(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	items := accounts()
	if id := r.URL.Query().Get("id"); id != "" {
		items = filterByString(items, "id", id)
	}
	writeEnvelope(w, http.StatusOK, items, nil, resultInfo(len(items)))
}

func (s *Server) handleAccountSubresources(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, "/accounts/"), "/")
	parts := strings.Split(rest, "/")
	if len(parts) < 2 {
		writeEnvelope(w, http.StatusNotFound, nil, []message{{Code: 1003, Message: "route not found"}}, nil)
		return
	}
	accountID := parts[0]
	switch parts[1] {
	case "logs":
		if len(parts) >= 3 && parts[2] == "audit" {
			items := auditLogs(accountID)
			writeEnvelope(w, http.StatusOK, items, nil, resultInfo(len(items)))
			return
		}
		writeEnvelope(w, http.StatusNotFound, nil, []message{{Code: 1003, Message: "route not found"}}, nil)
	case "rulesets":
		items := accountRulesets(accountID)
		writeEnvelope(w, http.StatusOK, items, nil, resultInfo(len(items)))
	case "waiting_rooms":
		items := accountWaitingRooms(accountID)
		writeEnvelope(w, http.StatusOK, items, nil, resultInfo(len(items)))
	case "workers":
		if len(parts) >= 3 && parts[2] == "scripts" {
			s.handleWorkers(w, accountID, parts[3:])
			return
		}
		writeEnvelope(w, http.StatusNotFound, nil, []message{{Code: 1003, Message: "route not found"}}, nil)
	case "storage":
		if len(parts) >= 4 && parts[2] == "kv" && parts[3] == "namespaces" {
			s.handleKVNamespaces(w, accountID, parts[4:])
			return
		}
		writeEnvelope(w, http.StatusNotFound, nil, []message{{Code: 1003, Message: "route not found"}}, nil)
	case "r2":
		if len(parts) >= 3 && parts[2] == "buckets" {
			s.handleR2Buckets(w, accountID, parts[3:])
			return
		}
		writeEnvelope(w, http.StatusNotFound, nil, []message{{Code: 1003, Message: "route not found"}}, nil)
	default:
		writeEnvelope(w, http.StatusNotFound, nil, []message{{Code: 1003, Message: "route not found"}}, nil)
	}
}

func (s *Server) handleGraphQL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"errors": []map[string]any{{"message": "method not allowed"}}})
		return
	}
	writeJSON(w, http.StatusOK, graphQLTrafficResponse())
}

func (s *Server) handleWorkers(w http.ResponseWriter, accountID string, parts []string) {
	if len(parts) == 0 || parts[0] == "" {
		items := workers(accountID)
		writeEnvelope(w, http.StatusOK, items, nil, resultInfo(len(items)))
		return
	}
	scriptName := parts[0]
	if len(parts) == 2 && parts[1] == "subdomain" {
		subdomain, ok := workerSubdomain(accountID, scriptName)
		if !ok {
			writeEnvelope(w, http.StatusNotFound, nil, []message{{Code: 1004, Message: "worker not found"}}, nil)
			return
		}
		writeEnvelope(w, http.StatusOK, subdomain, nil, nil)
		return
	}
	if len(parts) == 2 && parts[1] == "versions" {
		items := workerVersions(accountID, scriptName)
		writeEnvelope(w, http.StatusOK, items, nil, resultInfo(len(items)))
		return
	}
	writeEnvelope(w, http.StatusNotFound, nil, []message{{Code: 1003, Message: "route not found"}}, nil)
}

func (s *Server) handleKVNamespaces(w http.ResponseWriter, accountID string, parts []string) {
	if len(parts) == 0 || parts[0] == "" {
		items := kvNamespaces(accountID)
		writeEnvelope(w, http.StatusOK, items, nil, resultInfo(len(items)))
		return
	}
	namespace, ok := kvNamespace(accountID, parts[0])
	if !ok {
		writeEnvelope(w, http.StatusNotFound, nil, []message{{Code: 1005, Message: "namespace not found"}}, nil)
		return
	}
	writeEnvelope(w, http.StatusOK, namespace, nil, nil)
}

func (s *Server) handleR2Buckets(w http.ResponseWriter, accountID string, parts []string) {
	if len(parts) == 0 || parts[0] == "" {
		items := r2Buckets(accountID)
		writeEnvelope(w, http.StatusOK, map[string]any{"buckets": items}, nil, nil)
		return
	}
	bucket, ok := r2Bucket(accountID, parts[0])
	if !ok {
		writeEnvelope(w, http.StatusNotFound, nil, []message{{Code: 1006, Message: "bucket not found"}}, nil)
		return
	}
	writeEnvelope(w, http.StatusOK, bucket, nil, nil)
}

func (s *Server) handleZones(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	items := zones()
	if accountID := r.URL.Query().Get("account.id"); accountID != "" {
		items = filterByNestedString(items, "account", "id", accountID)
	}
	if name := r.URL.Query().Get("name"); name != "" {
		items = filterByString(items, "name", name)
	}
	if status := r.URL.Query().Get("status"); status != "" {
		items = filterByString(items, "status", status)
	}
	writeEnvelope(w, http.StatusOK, items, nil, resultInfo(len(items)))
}

func (s *Server) handleZoneSubresources(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	rest := strings.TrimPrefix(r.URL.Path, "/zones/")
	if strings.Contains(rest, "/settings/") {
		parts := strings.SplitN(rest, "/settings/", 2)
		setting, ok := zoneSetting(parts[0], strings.Trim(parts[1], "/"))
		if !ok {
			writeEnvelope(w, http.StatusNotFound, nil, []message{{Code: 1007, Message: "setting not found"}}, nil)
			return
		}
		writeEnvelope(w, http.StatusOK, setting, nil, nil)
		return
	}
	if strings.Contains(rest, "/cache/") {
		parts := strings.SplitN(rest, "/cache/", 2)
		setting, ok := cacheSetting(parts[0], strings.Trim(parts[1], "/"))
		if !ok {
			writeEnvelope(w, http.StatusNotFound, nil, []message{{Code: 1008, Message: "cache setting not found"}}, nil)
			return
		}
		writeEnvelope(w, http.StatusOK, setting, nil, nil)
		return
	}
	if strings.HasSuffix(rest, "/dns_records") {
		zoneID := strings.TrimSuffix(rest, "/dns_records")
		zoneID = strings.TrimSuffix(zoneID, "/")
		records := dnsRecords(zoneID)
		if recordType := r.URL.Query().Get("type"); recordType != "" {
			records = filterByString(records, "type", recordType)
		}
		if name := r.URL.Query().Get("name"); name != "" {
			records = filterByString(records, "name", name)
		}
		if content := r.URL.Query().Get("content"); content != "" {
			records = filterByString(records, "content", content)
		}
		writeEnvelope(w, http.StatusOK, records, nil, resultInfo(len(records)))
		return
	}
	if strings.HasSuffix(rest, "/rulesets") {
		zoneID := strings.TrimSuffix(rest, "/rulesets")
		zoneID = strings.TrimSuffix(zoneID, "/")
		items := zoneRulesets(zoneID)
		writeEnvelope(w, http.StatusOK, items, nil, resultInfo(len(items)))
		return
	}
	if strings.Contains(rest, "/waiting_rooms/") {
		parts := strings.SplitN(rest, "/waiting_rooms/", 2)
		room, ok := waitingRoom(parts[0], strings.Trim(parts[1], "/"))
		if !ok {
			writeEnvelope(w, http.StatusNotFound, nil, []message{{Code: 1010, Message: "waiting room not found"}}, nil)
			return
		}
		writeEnvelope(w, http.StatusOK, room, nil, nil)
		return
	}
	if strings.HasSuffix(rest, "/waiting_rooms") {
		zoneID := strings.TrimSuffix(rest, "/waiting_rooms")
		zoneID = strings.TrimSuffix(zoneID, "/")
		items := zoneWaitingRooms(zoneID)
		writeEnvelope(w, http.StatusOK, items, nil, resultInfo(len(items)))
		return
	}
	zoneID := strings.Trim(rest, "/")
	for _, zone := range zones() {
		if zone["id"] == zoneID {
			writeEnvelope(w, http.StatusOK, zone, nil, nil)
			return
		}
	}
	writeEnvelope(w, http.StatusNotFound, nil, []message{{Code: 7003, Message: "Could not route to /zones/" + zoneID}}, nil)
}

func hasBearerToken(r *http.Request) bool {
	header := r.Header.Get("Authorization")
	return strings.HasPrefix(header, "Bearer ") && strings.TrimPrefix(header, "Bearer ") != ""
}

func requireGet(w http.ResponseWriter, r *http.Request) bool {
	if r.Method == http.MethodGet {
		return true
	}
	writeEnvelope(w, http.StatusMethodNotAllowed, nil, []message{{Code: 1001, Message: "method not allowed"}}, nil)
	return false
}

type message struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

type info struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Count      int `json:"count"`
	TotalCount int `json:"total_count"`
	TotalPages int `json:"total_pages"`
}

func resultInfo(count int) *info {
	return &info{Page: 1, PerPage: 20, Count: count, TotalCount: count, TotalPages: 1}
}

func writeEnvelope(w http.ResponseWriter, status int, result any, errors []message, resultInfo *info) {
	success := status < 400 && len(errors) == 0
	if errors == nil {
		errors = []message{}
	}
	writeJSON(w, status, map[string]any{
		"success":     success,
		"result":      result,
		"errors":      errors,
		"messages":    []message{},
		"result_info": resultInfo,
	})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(data)
}
