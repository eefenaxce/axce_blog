package service

import (
	"context"
	"errors"
	"net/mail"
	"strings"

	"github.com/eefenaxce/axce_blog/internal/models"
	"github.com/eefenaxce/axce_blog/internal/repository"
)

var (
	ErrCommentNotFound  = errors.New("comment not found")
	ErrCommentsDisabled = errors.New("comments are disabled")
	ErrInvalidComment   = errors.New("invalid comment request")
)

type CommentService struct {
	commentRepo    repository.CommentRepository
	articleRepo    repository.ArticleRepository
	settingService *SettingService
}

func NewCommentService(commentRepo repository.CommentRepository, articleRepo repository.ArticleRepository, settingService *SettingService) *CommentService {
	return &CommentService{
		commentRepo:    commentRepo,
		articleRepo:    articleRepo,
		settingService: settingService,
	}
}

type CreateCommentInput struct {
	ArticleID    int
	ParentID     *int
	UserID       *int
	AuthorName   string
	AuthorEmail  string
	AuthorURL    string
	AuthorAvatar string
	Content      string
	IPAddress    string
}

func (s *CommentService) Create(ctx context.Context, input CreateCommentInput) (*models.Comment, error) {
	enabled, _ := s.settingService.Get(ctx, "enable_comments")
	if enabled == "false" {
		return nil, ErrCommentsDisabled
	}

	if strings.TrimSpace(input.Content) == "" {
		return nil, ErrInvalidComment
	}

	article, err := s.articleRepo.GetByID(ctx, input.ArticleID)
	if err != nil {
		return nil, ErrArticleNotFound
	}
	if !article.CommentEnabled {
		return nil, ErrCommentsDisabled
	}

	status := "approved"
	requireReview, _ := s.settingService.Get(ctx, "require_review")
	if requireReview == "true" {
		status = "pending"
	}

	authorName := strings.TrimSpace(input.AuthorName)
	authorEmail := strings.TrimSpace(input.AuthorEmail)
	if input.UserID == nil {
		if authorName == "" {
			return nil, errors.New("nickname is required")
		}
		if authorEmail != "" {
			if _, err := mail.ParseAddress(authorEmail); err != nil {
				return nil, errors.New("invalid email address")
			}
		}
	}

	comment := &models.Comment{
		ArticleID:    input.ArticleID,
		ParentID:     input.ParentID,
		UserID:       input.UserID,
		AuthorName:   authorName,
		AuthorEmail:  authorEmail,
		AuthorURL:    strings.TrimSpace(input.AuthorURL),
		AuthorAvatar: strings.TrimSpace(input.AuthorAvatar),
		Content:      strings.TrimSpace(input.Content),
		Status:       status,
		IPAddress:    input.IPAddress,
	}

	if err := s.commentRepo.Create(ctx, comment); err != nil {
		return nil, err
	}

	return comment, nil
}

func (s *CommentService) Delete(ctx context.Context, id int, userID int, isAdmin bool) error {
	comment, err := s.commentRepo.GetByID(ctx, id)
	if err != nil {
		return ErrCommentNotFound
	}

	if !isAdmin && (comment.UserID == nil || *comment.UserID != userID) {
		return errors.New("unauthorized to delete this comment")
	}

	return s.commentRepo.Delete(ctx, id)
}

func (s *CommentService) ListByArticle(ctx context.Context, articleID int, offset, limit int) ([]*models.Comment, int, error) {
	return s.commentRepo.ListByArticle(ctx, articleID, offset, limit)
}

func (s *CommentService) List(ctx context.Context, offset, limit int, status string) ([]*models.Comment, int, error) {
	return s.commentRepo.List(ctx, offset, limit, status)
}

func (s *CommentService) Approve(ctx context.Context, id int) error {
	return s.commentRepo.UpdateStatus(ctx, id, "approved")
}

func (s *CommentService) Reject(ctx context.Context, id int) error {
	return s.commentRepo.UpdateStatus(ctx, id, "rejected")
}

// Count returns the total number of approved comments.
func (s *CommentService) Count(ctx context.Context) (int, error) {
	_, total, err := s.commentRepo.List(ctx, 0, 1, "")
	return total, err
}

// CountByArticles returns approved comment counts for a set of articles.
func (s *CommentService) CountByArticles(ctx context.Context, articleIDs []int) (map[int]int, error) {
	return s.commentRepo.CountByArticles(ctx, articleIDs)
}
