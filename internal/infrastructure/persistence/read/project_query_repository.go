package read

import (
	"context"
	"errors"
	"time"

	"go-api/internal/domain/paginate"
	domainproject "go-api/internal/domain/project"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type projectReadRepository struct {
	db *gorm.DB
}

func NewProjectReadRepository(db *gorm.DB) domainproject.ReadRepository {
	return &projectReadRepository{db: db}
}

type projectScan struct {
	ID        uuid.UUID `gorm:"column:id"`
	Name      string    `gorm:"column:name"`
	Status    string    `gorm:"column:status"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (r *projectReadRepository) FindByID(ctx context.Context, id uuid.UUID) (*domainproject.View, error) {
	var m projectScan
	err := r.db.WithContext(ctx).Table("project").Where("id = ?", id).First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return scanProjectView(m), nil
}

func (r *projectReadRepository) List(ctx context.Context, query paginate.PaginateQuery) ([]domainproject.View, int64, error) {
	db := r.db.WithContext(ctx).Table("project")
	if query.Search != "" {
		db = db.Where("name ILIKE ?", "%"+query.Search+"%")
	}
	scoped, total, err := Paginate(db, query)
	if err != nil {
		return nil, 0, err
	}
	var rows []projectScan
	if err := scoped.Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	views := make([]domainproject.View, 0, len(rows))
	for _, row := range rows {
		views = append(views, *scanProjectView(row))
	}
	return views, total, nil
}

func scanProjectView(m projectScan) *domainproject.View {
	return &domainproject.View{
		ID:        m.ID,
		Name:      m.Name,
		Status:    domainproject.Status(m.Status),
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}
