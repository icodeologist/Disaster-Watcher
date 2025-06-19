package helpers

import (
	"github.com/icodeologist/disasterwatch/models"
)

type Stack struct {
	elements []*models.Report
}

func (s *Stack) Push(report *models.Report) {
	s.elements = append(s.elements, report)
}

func (s *Stack) Pop() (*models.Report, bool) {
	if len(s.elements) == 0 {
		return nil, false
	}

	idx := len(s.elements) - 1
	report := s.elements[idx]
	s.elements = s.elements[:idx]
	return report, true
}
