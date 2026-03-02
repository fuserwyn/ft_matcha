package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"matcha/api/internal/middleware"
	"matcha/api/internal/repository"
)

type ReportsHandler struct {
	reports  *repository.ReportRepository
	userRepo *repository.UserRepository
}

func NewReportsHandler(reports *repository.ReportRepository, userRepo *repository.UserRepository) *ReportsHandler {
	return &ReportsHandler{reports: reports, userRepo: userRepo}
}

type ReportUserReq struct {
	Reason  string  `json:"reason" binding:"required"`
	Comment *string `json:"comment"`
}

var allowedReportReasons = map[string]struct{}{
	"fake_account":  {},
	"spam":          {},
	"harassment":    {},
	"inappropriate": {},
	"scam":          {},
	"other":         {},
}

// ReportUser godoc
// @Summary	Report fake or abusive account
// @Tags		reports
// @Security	BearerAuth
// @Accept		json
// @Produce	json
// @Param		id		path		string			true	"Target user ID"
// @Param		body	body		ReportUserReq	true	"Report payload"
// @Success	201		{object}	map[string]interface{}
// @Failure	400		{object}	map[string]string
// @Failure	404		{object}	map[string]string
// @Router		/api/v1/users/{id}/report [post]
func (h *ReportsHandler) ReportUser(c *gin.Context) {
	reporterID := c.MustGet(middleware.UserIDKey).(uuid.UUID)
	targetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if reporterID == targetID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot report yourself"})
		return
	}
	if u, err := h.userRepo.GetByID(c.Request.Context(), targetID); err != nil || u == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	var req ReportUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	reason := strings.ToLower(strings.TrimSpace(req.Reason))
	if _, ok := allowedReportReasons[reason]; !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid reason"})
		return
	}
	if req.Comment != nil {
		comment := strings.TrimSpace(*req.Comment)
		if len(comment) > 500 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "comment: max 500 characters"})
			return
		}
		req.Comment = &comment
	}

	report, err := h.reports.Upsert(c.Request.Context(), reporterID, targetID, reason, req.Comment)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":               report.ID,
		"reporter_user_id": report.ReporterUserID,
		"target_user_id":   report.TargetUserID,
		"reason":           report.Reason,
		"comment":          report.Comment,
		"status":           report.Status,
		"created_at":       report.CreatedAt,
		"updated_at":       report.UpdatedAt,
	})
}

// ListMyReports godoc
// @Summary	List my submitted reports
// @Tags		reports
// @Security	BearerAuth
// @Produce	json
// @Param		limit	query		int	false	"Limit (default 20)"
// @Param		offset	query		int	false	"Offset"
// @Success	200		{array}		object
// @Router		/api/v1/reports/me [get]
func (h *ReportsHandler) ListMyReports(c *gin.Context) {
	reporterID := c.MustGet(middleware.UserIDKey).(uuid.UUID)
	limit, offset := parseLimitOffset(c)

	reports, err := h.reports.ListByReporter(c.Request.Context(), reporterID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resp := make([]gin.H, len(reports))
	for i := range reports {
		resp[i] = gin.H{
			"id":               reports[i].ID,
			"reporter_user_id": reports[i].ReporterUserID,
			"target_user_id":   reports[i].TargetUserID,
			"reason":           reports[i].Reason,
			"comment":          reports[i].Comment,
			"status":           reports[i].Status,
			"created_at":       reports[i].CreatedAt,
			"updated_at":       reports[i].UpdatedAt,
		}
	}
	c.JSON(http.StatusOK, resp)
}
