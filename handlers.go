package main

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Health Check

func (s *CortexStore) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":    "ok",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// Cortex API Handlers

func (s *CortexStore) handleRemember(w http.ResponseWriter, r *http.Request) {
	var req RememberRequest
	if err := parseJSONBody(r, &req); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Content) == "" {
		http.Error(w, "content is required", http.StatusBadRequest)
		return
	}

	if req.Type == "" {
		req.Type = DefaultMemType
	}
	if req.Importance == 0 {
		req.Importance = DefaultImportance
	}

	mem := Memory{
		Type:       req.Type,
		Content:    req.Content,
		Entity:     req.Entity,
		Tags:       req.Tags,
		Importance: req.Importance,
	}

	if err := s.CreateMemory(&mem); err != nil {
		slog.Error("remember insert error", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, RememberResponse{ID: mem.ID})
}

func (s *CortexStore) handleRecall(w http.ResponseWriter, r *http.Request) {
	query := getQueryParam(r, "q")
	memType := getQueryParam(r, "type")
	limit := parseLimit(getQueryParam(r, "limit"), DefaultLimit, MaxLimit)

	memories, err := s.SearchMemories(query, memType, limit)
	if err != nil {
		slog.Error("recall query error", "error", err, "query", query)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	mapMemoryMetadata(memories)
	writeJSON(w, http.StatusOK, memories)
}

func (s *CortexStore) handleSetFact(w http.ResponseWriter, r *http.Request) {
	entity := getQueryParam(r, "entity")
	if entity == "" {
		http.Error(w, "entity is required (query param)", http.StatusBadRequest)
		return
	}

	var req FactRequest
	if err := parseJSONBody(r, &req); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Key) == "" {
		http.Error(w, "key is required", http.StatusBadRequest)
		return
	}

	ent, err := s.GetEntity(entity)
	data := map[string]any{}
	if err == nil && ent.Data != "" {
		data = unmarshalEntityData(ent.Data)
	} else if err != nil && err != gorm.ErrRecordNotFound {
		slog.Error("get entity error", "error", err, "entity", entity)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	data[req.Key] = req.Value
	ent = &Entity{
		Name:      entity,
		Data:      marshalEntityData(data),
		UpdatedAt: time.Now(),
	}

	if err := s.CreateOrUpdateEntity(ent); err != nil {
		slog.Error("set fact error", "error", err, "entity", entity)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *CortexStore) handleGetEntity(w http.ResponseWriter, r *http.Request) {
	name := getQueryParam(r, "name")
	if name == "" {
		http.Error(w, "name is required (query param)", http.StatusBadRequest)
		return
	}

	ent, err := s.GetEntity(name)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		slog.Error("get entity error", "error", err, "name", name)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	mapEntityDataSingle(ent)
	writeJSON(w, http.StatusOK, ent)
}

func (s *CortexStore) handleListEntities(w http.ResponseWriter, r *http.Request) {
	entities, err := s.ListEntities()
	if err != nil {
		slog.Error("list entities error", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	mapEntityData(entities)
	writeJSON(w, http.StatusOK, entities)
}

func (s *CortexStore) handleAddRelation(w http.ResponseWriter, r *http.Request) {
	var req RelationRequest
	if err := parseJSONBody(r, &req); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.From) == "" || strings.TrimSpace(req.To) == "" || strings.TrimSpace(req.Type) == "" {
		http.Error(w, "from, to and type are required", http.StatusBadRequest)
		return
	}

	rel := Relation{
		From: req.From,
		To:   req.To,
		Type: req.Type,
	}

	if err := s.CreateOrUpdateRelation(&rel); err != nil {
		slog.Error("add relation error", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *CortexStore) handleListRelations(w http.ResponseWriter, r *http.Request) {
	entity := getQueryParam(r, "entity")
	relations, err := s.GetRelations(entity)
	if err != nil {
		slog.Error("list relations error", "error", err, "entity", entity)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, relations)
}

func (s *CortexStore) handleStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.GetStats()
	if err != nil {
		slog.Error("stats error", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, stats)
}

// Neutron-compatible Seeds API Handlers

func (s *CortexStore) handleStoreSeed(w http.ResponseWriter, r *http.Request) {
	var req StoreSeedRequest
	if err := parseJSONBody(r, &req); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}

	fields := map[string]string{
		"appId":         req.AppID,
		"externalUserId": req.ExternalUserID,
		"content":       req.Content,
	}
	if field, ok := validateRequired(fields); !ok {
		http.Error(w, "missing required field: "+field, http.StatusBadRequest)
		return
	}

	mem := Memory{
		Type:          DefaultMemType,
		Content:       req.Content,
		AppID:         req.AppID,
		ExternalUserID: req.ExternalUserID,
		Metadata:      marshalMetadata(req.Metadata),
		Importance:    DefaultImportance,
	}

	if err := s.CreateMemory(&mem); err != nil {
		slog.Error("store seed error", "error", err, "appId", req.AppID, "userId", req.ExternalUserID)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, StoreSeedResponse{
		ID:      mem.ID,
		Message: "Memory stored successfully",
	})
}

func (s *CortexStore) handleQuerySeed(w http.ResponseWriter, r *http.Request) {
	var req QuerySeedRequest
	if err := parseJSONBody(r, &req); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}

	fields := map[string]string{
		"appId":         req.AppID,
		"externalUserId": req.ExternalUserID,
		"query":         req.Query,
	}
	if field, ok := validateRequired(fields); !ok {
		http.Error(w, "missing required field: "+field, http.StatusBadRequest)
		return
	}

	limit := req.Limit
	if limit <= 0 || limit > MaxLimit {
		limit = 5
	}

	memories, err := s.SearchMemoriesByTenant(req.AppID, req.ExternalUserID, req.Query, limit)
	if err != nil {
		slog.Error("query seed error", "error", err, "appId", req.AppID, "userId", req.ExternalUserID, "query", req.Query)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	results := make([]QuerySeedResult, 0, len(memories))
	for _, mem := range memories {
		metadata := unmarshalMetadata(mem.Metadata)

		similarity := 0.8
		if strings.Contains(strings.ToLower(mem.Content), strings.ToLower(req.Query)) {
			similarity = 0.95
		}

		results = append(results, QuerySeedResult{
			ID:         mem.ID,
			Content:    mem.Content,
			Metadata:   metadata,
			CreatedAt:  mem.CreatedAt,
			Similarity: similarity,
		})
	}

	writeJSON(w, http.StatusOK, results)
}

func (s *CortexStore) handleDeleteSeed(w http.ResponseWriter, r *http.Request) {
	idStr, err := extractPathID(r.URL.Path, "/seeds/")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id, err := parseID(idStr)
	if err != nil {
		http.Error(w, "invalid id format", http.StatusBadRequest)
		return
	}

	appID := getQueryParam(r, "appId")
	externalUserID := getQueryParam(r, "externalUserId")

	fields := map[string]string{
		"appId":         appID,
		"externalUserId": externalUserID,
	}
	if field, ok := validateRequired(fields); !ok {
		http.Error(w, "missing required query parameter: "+field, http.StatusBadRequest)
		return
	}

	mem, err := s.GetMemoryByIDAndTenant(id, appID, externalUserID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "Memory not found", http.StatusNotFound)
			return
		}
		slog.Error("delete seed error", "error", err, "id", id, "appId", appID, "userId", externalUserID)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := s.DeleteMemory(mem); err != nil {
		slog.Error("delete seed error", "error", err, "id", mem.ID)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, DeleteSeedResponse{
		Message: "Memory deleted successfully",
		ID:      mem.ID,
	})
}
