package excel

import (
	"context"
	"fmt"
	"github.com/360EntSecGroup-Skylar/excelize"
	"github.com/alexey-zayats/claim-parser/internal/dict"
	"github.com/alexey-zayats/claim-parser/internal/model"
	"github.com/alexey-zayats/claim-parser/internal/parser"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"regexp"
	"strings"
)

// Parser ...
type Parser struct {
	reNumber *regexp.Regexp
}

// Name ...
const Name = "excel"

// Register ...
func Register() {
	parser.Instance().Add(Name, NewParser)
}

// NewParser ...
func NewParser() (parser.Backend, error) {
	return &Parser{
		reNumber: regexp.MustCompile(`((?:\p{L}{1})\s?(?:\d{3})\s?(?:\p{L}{2})\s?(?:\d{2,3})\s?(?i:rus)?)\s?(?:.+)`),
	}, nil
}

// Parse ...
func (p *Parser) Parse(ctx context.Context, param *dict.Dict) (interface{}, error) {

	var path string

	if iface, ok := param.Get("path"); ok {
		path = iface.(string)
	} else {
		return nil, fmt.Errorf("not found 'path' in param dict")
	}

	logrus.WithFields(logrus.Fields{"name": Name, "path": path}).Debug("Parser.Parse")

	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, "unable open xlsx file %s", path)
	}

	sheetName := "Лист1"
	kind := f.GetCellValue(sheetName, "B5")
	address := f.GetCellValue(sheetName, "B7")

	source := f.GetCellValue(sheetName, "B12")

	company := &model.Claim{
		District: f.GetCellValue(sheetName, "A1"),
		Company: model.Company{
			Activity: strings.ReplaceAll(kind, "\n", ", "),
			Title:    f.GetCellValue(sheetName, "B6"),
			Address:  strings.ReplaceAll(address, "\n", ", "),
			INN:      strings.ReplaceAll(f.GetCellValue(sheetName, "B8"), " ", ""),
		},
		Cars:        p.parseCars(source),
		Agreement:   f.GetCellValue(sheetName, "B13"),
		Reliability: f.GetCellValue(sheetName, "B14"),
		Reason:      nil,
		Valid:       true,
		Source:      source,
	}

	company.Company.Head.Contact = model.Contact{
		Phone: f.GetCellValue(sheetName, "B10"),
		EMail: f.GetCellValue(sheetName, "B11"),
	}

	fio := strings.Split(f.GetCellValue(sheetName, "B9"), " ")

	if len(fio) < 3 {
		company.Valid = false
		reason := "Нет данных по ФИО руководителя"
		company.Reason = &reason
	} else {
		company.Company.Head.FIO = model.FIO{
			Surname:    fio[0],
			Name:       fio[1],
			Patronymic: fio[2],
		}
	}

	return company, nil
}

func (p *Parser) parseCars(data string) []model.Car {
	cars := make([]model.Car, 0)

	for _, item := range strings.Split(data, "\n") {
		if len(item) < 15 {
			continue
		}

		matches := p.reNumber.FindAllStringSubmatch(item, -1)
		if len(matches) > 0 {

			numberS := matches[0][1]
			fioS := strings.ReplaceAll(item, numberS, "")

			numberS = strings.TrimSpace(numberS)
			numberS = strings.ReplaceAll(numberS, " ", "")
			numberS = strings.ToUpper(numberS)

			re0 := regexp.MustCompile(`\d+\.`)
			fioS = re0.ReplaceAllString(fioS, "")

			if dashPos := strings.Index(fioS, "-"); dashPos > 0 {
				fioS = fioS[dashPos+1:]
			} else {
				fioS = strings.ReplaceAll(fioS, "-", "")
			}

			fioS = strings.ReplaceAll(fioS, ".", "")
			fioS = strings.TrimSpace(fioS)

			fio := regexp.MustCompile(`\s+`).Split(fioS, -1)

			car := model.Car{
				Number: numberS,
			}

			if len(fio) >= 3 {
				car.FIO.Surname = fio[0]
				car.FIO.Name = fio[1]
				car.FIO.Patronymic = fio[2]
				car.Valid = true
			} else {
				reason := "Нет данных по ФИО водителя"
				car.Reason = &reason
			}

			cars = append(cars, car)
		}
	}

	return cars
}
