package main

import (
	"github.com/xorcare/pointer"
	"testing_system/common/db/models"
	"testing_system/lib/customfields"
)

type XProblemXML struct {
	ShortName string `xml:"short-name,attr"`
	Revision  string `xml:"revision,attr"`
	Url       string `xml:"url,attr"`

	Names      XProblemNames      `xml:"names"`
	Statements XProblemStatements `xml:"statements"`

	Judging XJudging `xml:"judging"`
	Files   XFiles   `xml:"files"`
	Assets  XAssets  `xml:"assets"`
}

type XProblemNames struct {
	Names []*XProblemName `xml:"name"`
}

type XProblemName struct {
	Language string `xml:"language,attr"`
	Value    string `xml:"value,attr"`
}

type XProblemStatements struct {
	Statements []*XProblemStatement `xml:"statement"`
}

type XProblemStatement struct {
	Language string `xml:"language,attr"`
	Type     string `xml:"type,attr"`
	Path     string `xml:"path,attr"`
}

type XJudging struct {
	InputFile  string      `xml:"input-file,attr"`
	OutputFile string      `xml:"output-file,attr"`
	Testsets   []*XTestset `xml:"testset"`
}

type XTestset struct {
	Name        string  `xml:"name,attr"`
	TimeLimit   int     `xml:"time-limit"`
	MemoryLimit int     `xml:"memory-limit"`
	Tests       XTests  `xml:"tests"`
	Groups      XGroups `xml:"groups"`
}

type XTests struct {
	Tests []*XTest `xml:"test"`
}

type XTest struct {
	Group  string  `xml:"group,attr,omitempty"`
	Points float64 `xml:"points,attr,omitempty"`
	Sample bool    `xml:"sample,attr,omitempty"`
}

type XGroups struct {
	Groups []*XGroup `xml:"group"`
}

type XGroup struct {
	Name           string             `xml:"name,attr"`
	FeedbackPolicy string             `xml:"feedback-policy,attr"`
	PointsPolicy   string             `xml:"points-policy,attr"`
	Points         float64            `xml:"points,attr"`
	Dependencies   XGroupDependencies `xml:"dependencies"`
}

type XGroupDependencies struct {
	Dependencies []*XGroupDependency `xml:"dependency"`
}

type XGroupDependency struct {
	Group string `xml:"group,attr"`
}

type XFiles struct {
	Resources XResources `xml:"resources"`
}

type XResources struct {
	Resources []*XSource `xml:"file"`
}

type XAssets struct {
	Checker    XSourceFile  `xml:"checker"`
	Interactor *XSourceFile `xml:"interactor,omitempty"`
	Solutions  XSolutions   `xml:"solutions"`
}

type XSourceFile struct {
	Tag    string  `xml:"tag,attr,omitempty"`
	Source XSource `xml:"source"`
}

type XSource struct {
	Path string `xml:"path,attr"`
	Type string `xml:"type,attr"`
}

type XSolutions struct {
	Solutions []*XSourceFile `xml:"solution"`
}

func buildProblemModel() *models.Problem {
	prob := &models.Problem{
		Name: probXML.ShortName,
	}
	var testset *XTestset
	for _, someTestset := range probXML.Judging.Testsets {
		if someTestset.Name != "tests" {
			continue
		}
		testset = someTestset
	}
	if testset == nil {
		panic("no testset tests found")
	}
	prob.TimeLimit = customfields.Time(testset.TimeLimit * 1000 * 1000)
	prob.MemoryLimit = customfields.Memory(testset.MemoryLimit)
	prob.TestsNumber = uint64(len(testset.Tests.Tests))

	if len(testset.Groups.Groups) == 0 {
		prob.ProblemType = models.ProblemTypeICPC
		return prob
	}

	prob.ProblemType = models.ProblemTypeIOI
	for _, xGroup := range testset.Groups.Groups {
		group := &models.TestGroup{
			Name:       xGroup.Name,
			FirstTest:  prob.TestsNumber,
			GroupScore: pointer.Float64(0),
		}
		for _, dependency := range xGroup.Dependencies.Dependencies {
			group.RequiredGroupNames = append(group.RequiredGroupNames, dependency.Group)
		}
		switch xGroup.FeedbackPolicy {
		case "none":
			group.FeedbackType = models.TestGroupFeedbackTypeNone
		case "points":
			group.FeedbackType = models.TestGroupFeedbackTypePoints
		case "icpc":
			group.FeedbackType = models.TestGroupFeedbackTypeICPC
		case "complete":
			group.FeedbackType = models.TestGroupFeedbackTypeComplete
		default:
			panic("unknown feedback policy")
		}

		switch xGroup.PointsPolicy {
		case "complete-group":
			group.ScoringType = models.TestGroupScoringTypeComplete
		case "each-test":
			group.ScoringType = models.TestGroupScoringTypeEachTest
		default:
			panic("unknown points policy")
		}

		prob.TestGroups = append(prob.TestGroups, group)
	}

	for i, test := range testset.Tests.Tests {
		id := uint64(i + 1)
		if test.Group == "" {
			panic("test group is required for each test")
		}
		for _, group := range prob.TestGroups {
			if group.Name == test.Group {
				group.FirstTest = min(group.FirstTest, id)
				group.LastTest = max(group.LastTest, id)
				*group.GroupScore += test.Points
			}
		}
	}

	for _, group := range prob.TestGroups {
		if group.ScoringType == models.TestGroupScoringTypeEachTest {
			group.TestScore = pointer.Float64(*group.GroupScore / float64(group.LastTest-group.FirstTest+1))
			group.GroupScore = nil
		}
	}

	return prob
}
