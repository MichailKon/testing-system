package tsapi

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/xorcare/pointer"
	"gorm.io/gorm"
	"net/http"
	"strconv"
	"testing_system/common/connectors/storageconn"
	"testing_system/common/constants/resource"
	"testing_system/common/db/models"
)

type problemInList struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

type problemListFilter struct {
	Count int `form:"count" binding:"required"`
	Page  int `form:"page,default=1"`
}

func (h *Handler) getProblems(c *gin.Context) {
	filter := new(problemListFilter)
	if err := c.ShouldBindQuery(filter); err != nil {
		respError(c, http.StatusBadRequest, "%v", err)
		return
	}
	var problems []problemInList
	err := h.base.DB.
		WithContext(c).
		Model(&models.Problem{}).
		Limit(filter.Count).
		Offset((filter.Page - 1) * filter.Count).
		Order("id desc").
		Find(&problems).
		Error
	if err != nil {
		respServerError(c, "Can not load problems, error: %v", err)
		return
	}
	respSuccess(c, problems)
}

func (h *Handler) getProblem(c *gin.Context) {
	problem, ok := h.findProblem(c, c.Param("id"))
	if !ok {
		return
	}
	respSuccess(c, problem)
}

func (h *Handler) addProblem(c *gin.Context) {
	var problem models.Problem
	if err := c.BindJSON(&problem); err != nil {
		respError(c, http.StatusBadRequest, "%v", err)
		return
	}

	h.base.DB.WithContext(c).Create(&problem)
	respSuccess(c, problem)
}

func (h *Handler) modifyProblem(c *gin.Context) {
	oldProblem, ok := h.findProblem(c, c.Param("id"))
	if !ok {
		return
	}
	var problem models.Problem
	if err := c.BindJSON(&problem); err != nil {
		respError(c, http.StatusBadRequest, "%v", err)
		return
	}
	problem.ID = oldProblem.ID
	if !checkProblemIsOK(c, problem) {
		return
	}
	err := h.base.DB.WithContext(c).Save(&problem).Error
	if err != nil {
		respError(
			c, http.StatusInternalServerError, "Can not update problem %d, error: %v", problem.ID, err,
		)
		return
	}
	respSuccessEmpty(c)
}

func (h *Handler) problemTestResourceGetter(resourceType resource.Type) func(c *gin.Context) {
	return func(c *gin.Context) {
		problem, ok := h.findProblem(c, c.Param("id"))
		if !ok {
			return
		}
		testID, ok := h.getProblemTestID(c, problem)
		if !ok {
			return
		}

		resp := h.base.StorageConnection.Download(&storageconn.Request{
			Resource:      resourceType,
			ProblemID:     uint64(problem.ID),
			TestID:        testID,
			DownloadBytes: true,
			DownloadHead:  pointer.Int64(h.config.LoadFilesHead),
			Ctx:           c,
		})
		if resp.Error != nil {
			if errors.Is(resp.Error, storageconn.ErrStorageFileNotFound) {
				respError(c, http.StatusNotFound, "%v for problem %d test %d does not exist", resourceType, problem.ID, testID)
				return
			}
			respServerError(
				c, "Can not load problem %d %v, error: %v", problem.ID, resourceType, resp.Error,
			)
			return
		}

		respSuccess(c, fileData{
			Filename: resp.Filename,
			Data:     string(resp.RawData),
			Size:     resp.Size,
		})
	}
}

type testIDHolder struct {
	Test uint64 `uri:"test" binding:"required"`
}

func (h *Handler) getProblemTestID(c *gin.Context, problem *models.Problem) (uint64, bool) {
	testID := new(testIDHolder)
	err := c.ShouldBindUri(testID)
	if err != nil {
		return 0, false
	}

	if testID.Test <= 0 || testID.Test > problem.TestsNumber {
		respError(c, http.StatusBadRequest, "Problem %d does not has test %d", problem.ID, problem.TestsNumber)
		return 0, false
	}
	return testID.Test, true
}

func (h *Handler) findProblemByID(c *gin.Context, id uint) (*models.Problem, bool) {
	problem := new(models.Problem)
	err := h.base.DB.
		WithContext(c).
		First(problem, id).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respError(c, http.StatusNotFound, "Problem with id %d not found", id)
		} else {
			respServerError(c, "Can not load problem %d, error: %v", id, err)
		}
		return nil, false
	}
	return problem, true
}

func (h *Handler) findProblem(c *gin.Context, id string) (*models.Problem, bool) {
	problemID, err := strconv.Atoi(id)
	if err != nil {
		respError(c, http.StatusBadRequest, "Can not parse problem id %s, error: %v", id, err)
		return nil, false
	}

	return h.findProblemByID(c, uint(problemID))
}

func checkProblemIsOK(c *gin.Context, problem models.Problem) bool {
	switch problem.ProblemType {
	case models.ProblemTypeICPC:
		return true
	case models.ProblemTypeIOI:
		lastTest := uint64(0)
		usedGroupNames := make(map[string]struct{})
		for _, group := range problem.TestGroups {
			_, ok := usedGroupNames[group.Name]
			if ok {
				respError(c, http.StatusBadRequest, "Group %s is used more than once", group.Name)
				return false
			}

			if group.FirstTest != lastTest+1 {
				respError(c, http.StatusBadRequest,
					"Group %s first test is incorrect, should be previous group last test + 1", group.Name,
				)
				return false
			}
			if group.FirstTest > group.LastTest {
				respError(c, http.StatusBadRequest, "Group %s first test is greater than last test", group.Name)
			}
			lastTest = group.LastTest
			switch group.ScoringType {
			case models.TestGroupScoringTypeComplete, models.TestGroupScoringTypeMin:
				if group.GroupScore == nil {
					respError(c, http.StatusBadRequest, "Group %s has no group score", group.Name)
					return false
				}
			case models.TestGroupScoringTypeEachTest:
				if group.TestScore == nil {
					respError(c, http.StatusBadRequest, "Group %s has no test score", group.Name)
					return false
				}
			default:
				respError(c, http.StatusBadRequest, "Group %s has invalid scoring type", group.Name)
				return false
			}

			switch group.FeedbackType {
			case models.TestGroupFeedbackTypeNone,
				models.TestGroupFeedbackTypePoints,
				models.TestGroupFeedbackTypeICPC,
				models.TestGroupFeedbackTypeComplete,
				models.TestGroupFeedbackTypeFull:
				// skip
			default:
				respError(c, http.StatusBadRequest, "Group %s has invalid feedback type", group.Name)
				return false
			}

			usedRequiredGroups := make(map[string]struct{})
			for _, required := range group.RequiredGroupNames {
				if _, ok = usedGroupNames[required]; !ok {
					respError(c, http.StatusBadRequest,
						"Group %s has required group name %s which is not present", group.Name, required,
					)
					return false
				}
				if _, ok = usedRequiredGroups[required]; ok {
					respError(c, http.StatusBadRequest,
						"Group %s has duplicate required group name %s", group.Name, required,
					)
				}
			}

			usedGroupNames[group.Name] = struct{}{}
		}
		if lastTest != problem.TestsNumber {
			respError(c, http.StatusBadRequest, "Not all tests are present in all groups")
			return false
		}
		return true
	default:
		respError(c, http.StatusBadRequest, "Invalid problem type")
		return false
	}
}
