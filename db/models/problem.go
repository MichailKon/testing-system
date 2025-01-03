package models

import "gorm.io/gorm"

type Problem struct {
	gorm.Model
	ProblemConfigId int
	ProblemConfig   ProblemConfig
}
