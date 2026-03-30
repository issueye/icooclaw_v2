package storage

import (
	"fmt"

	"gorm.io/gorm"
)

func pageQuery[T any](qry *gorm.DB, order string, page Page, records *[]T, label string) (Page, error) {
	if order != "" {
		qry = qry.Order(order)
	}

	if err := qry.Count(&page.Total).Error; err != nil {
		return page, fmt.Errorf("failed to count %s: %w", label, err)
	}

	if page.Page == 0 || page.Size == 0 {
		if err := qry.Find(records).Error; err != nil {
			return page, fmt.Errorf("failed to get %s: %w", label, err)
		}
		return page, nil
	}

	if err := qry.Limit(page.Size).
		Offset((page.Page - 1) * page.Size).
		Find(records).Error; err != nil {
		return page, fmt.Errorf("failed to get %s: %w", label, err)
	}

	return page, nil
}

func listOrdered[T any](qry *gorm.DB, order string, out *[]*T, label string) error {
	if order != "" {
		qry = qry.Order(order)
	}
	if err := qry.Find(out).Error; err != nil {
		return fmt.Errorf("failed to list %s: %w", label, err)
	}
	return nil
}

func deleteByField[T any](db *gorm.DB, field string, value any, model *T, label string) error {
	if err := db.Where(field+" = ?", value).Delete(model).Error; err != nil {
		return fmt.Errorf("failed to delete %s: %w", label, err)
	}
	return nil
}
