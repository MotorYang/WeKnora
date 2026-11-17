package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Tencent/WeKnora/internal/application/access"
	"github.com/Tencent/WeKnora/internal/middleware"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type embedKnowledgeConfigStub struct {
	ids []string
}

func (s embedKnowledgeConfigStub) PublicConfig(
	_ context.Context,
	_ *types.EmbedChannel,
) types.EmbedChannelPublicConfig {
	return types.EmbedChannelPublicConfig{KnowledgeBaseIDs: s.ids}
}

type embedKnowledgeLookupStub struct {
	kb *types.KnowledgeBase
}

func (s embedKnowledgeLookupStub) GetKnowledgeBaseByIDOnly(
	_ context.Context,
	_ string,
) (*types.KnowledgeBase, error) {
	return s.kb, nil
}

func (s embedKnowledgeLookupStub) GetKnowledgeBasesByIDsOnly(
	_ context.Context,
	_ []string,
) ([]*types.KnowledgeBase, error) {
	if s.kb == nil {
		return nil, nil
	}
	return []*types.KnowledgeBase{s.kb}, nil
}

func runEmbedKnowledgeUploadScope(
	t *testing.T,
	channel *types.EmbedChannel,
	allowed []string,
	kb *types.KnowledgeBase,
	requestedKB string,
) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST(
		"/api/v1/embed/:channel_id/knowledge-bases/:id/knowledge/file",
		func(c *gin.Context) {
			ctx := context.WithValue(c.Request.Context(), types.EmbedChannelContextKey, channel)
			ctx = types.WithCaller(ctx, types.Caller{
				TenantID: channel.TenantID,
				UserID:   "embed-channel",
				Role:     types.TenantRoleViewer,
			})
			c.Request = c.Request.WithContext(ctx)
			c.Next()
		},
		embedKnowledgeUploadScope(
			embedKnowledgeConfigStub{ids: allowed},
			embedKnowledgeLookupStub{kb: kb},
			"id",
		),
		func(c *gin.Context) {
			ctx := c.Request.Context()
			grant, ok := middleware.KBAccessFromContext(c)
			require.True(t, ok)
			require.Equal(t, requestedKB, grant.KnowledgeBase.ID)
			require.True(t, access.HasKBGrant(ctx, requestedKB, channel.TenantID, types.OrgRoleEditor))
			require.False(t, access.HasKBGrant(ctx, requestedKB, channel.TenantID, types.OrgRoleAdmin))
			require.Equal(t, types.TenantRoleViewer, types.CallerFromContext(ctx).Role)
			c.Status(http.StatusNoContent)
		},
	)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/embed/channel/knowledge-bases/"+requestedKB+"/knowledge/file",
		nil,
	)
	r.ServeHTTP(w, req)
	return w
}

func TestEmbedKnowledgeDocumentParamKeepsKBGrantAndPromotesDocumentID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	channel := &types.EmbedChannel{ID: "channel", TenantID: 7, AllowFileUpload: true}
	kb := &types.KnowledgeBase{ID: "kb-allowed", TenantID: 7}
	r := gin.New()
	r.PUT(
		"/api/v1/embed/:channel_id/knowledge-bases/:id/documents/:knowledge_id",
		func(c *gin.Context) {
			ctx := context.WithValue(c.Request.Context(), types.EmbedChannelContextKey, channel)
			ctx = types.WithCaller(ctx, types.Caller{TenantID: 7, UserID: "embed-channel", Role: types.TenantRoleViewer})
			c.Request = c.Request.WithContext(ctx)
			c.Next()
		},
		embedKnowledgeUploadScope(
			embedKnowledgeConfigStub{ids: []string{"kb-allowed"}},
			embedKnowledgeLookupStub{kb: kb},
			"id",
		),
		embedKnowledgeDocumentParam("knowledge_id"),
		func(c *gin.Context) {
			require.Equal(t, "document-1", c.Param("id"))
			require.True(t, access.HasKBGrant(c.Request.Context(), "kb-allowed", 7, types.OrgRoleEditor))
			c.Status(http.StatusNoContent)
		},
	)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/embed/channel/knowledge-bases/kb-allowed/documents/document-1", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNoContent, w.Code)
}

func TestEmbedKnowledgeUploadScopeGrantsOnlySelectedKnowledgeBase(t *testing.T) {
	channel := &types.EmbedChannel{ID: "channel", TenantID: 7, AllowFileUpload: true}
	kb := &types.KnowledgeBase{ID: "kb-allowed", TenantID: 7}
	w := runEmbedKnowledgeUploadScope(t, channel, []string{"kb-allowed"}, kb, "kb-allowed")
	require.Equal(t, http.StatusNoContent, w.Code)
}

func TestEmbedKnowledgeUploadScopeRejectsDisabledUpload(t *testing.T) {
	channel := &types.EmbedChannel{ID: "channel", TenantID: 7, AllowFileUpload: false}
	kb := &types.KnowledgeBase{ID: "kb-allowed", TenantID: 7}
	w := runEmbedKnowledgeUploadScope(t, channel, []string{"kb-allowed"}, kb, "kb-allowed")
	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestEmbedKnowledgeUploadScopeRejectsKnowledgeBaseOutsideChannel(t *testing.T) {
	channel := &types.EmbedChannel{ID: "channel", TenantID: 7, AllowFileUpload: true}
	kb := &types.KnowledgeBase{ID: "kb-other", TenantID: 7}
	w := runEmbedKnowledgeUploadScope(t, channel, []string{"kb-allowed"}, kb, "kb-other")
	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestEmbedKnowledgeUploadScopeRejectsCrossTenantKnowledgeBase(t *testing.T) {
	channel := &types.EmbedChannel{ID: "channel", TenantID: 7, AllowFileUpload: true}
	kb := &types.KnowledgeBase{ID: "kb-allowed", TenantID: 8}
	w := runEmbedKnowledgeUploadScope(t, channel, []string{"kb-allowed"}, kb, "kb-allowed")
	require.Equal(t, http.StatusNotFound, w.Code)
}
