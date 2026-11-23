package models

import "errors"

var (
	ErrReviewCaseAssigned = errors.New("review case is assigned to another analyst")
	ErrReviewCaseClosed   = errors.New("review case is closed")
)
