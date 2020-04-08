package fsdump

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/alexey-zayats/claim-parser/internal/dict"
	"github.com/alexey-zayats/claim-parser/internal/formstruct"
	"github.com/alexey-zayats/claim-parser/internal/model"
	"github.com/alexey-zayats/claim-parser/internal/parser"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"regexp"
	"strings"
)

// Parser ...
type Parser struct {
}

// Name ...
const Name = "fsdump"

// Register ...
func Register() {
	parser.Instance().Add(Name, NewParser)
}

// NewParser ...
func NewParser() (parser.Backend, error) {
	return &Parser{}, nil
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

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, "unable read file %s", path)
	}

	head := regexp.MustCompile(`/\* \d+ createdAt:(.[^\*]+)\*/`)
	nl := regexp.MustCompile(`\r?\n`)
	space := regexp.MustCompile(`^\s+(?:.+)\s+$`)
	object := regexp.MustCompile(`ObjectId\("(.[^"]+)"\)`)

	inClaim := false

	var lines []string
	var created string

	var claims []*model.Claim

	for _, line := range nl.Split(string(data), -1) {

		line = space.ReplaceAllString(line, "")

		if head.MatchString(line) {
			inClaim = true
			lines = make([]string, 0)

			m := head.FindAllStringSubmatch(line, -1)
			if len(m) > 0 {
				created = fmt.Sprintf("\t\"createdAt\" : \"%s\",", m[0][1])
			}

			continue
		}

		if len(line) == 0 {

			inClaim = false

			last := len(lines) - 1
			lines = append(lines, lines[last])
			copy(lines[3:], lines[2:last])
			lines[2] = created

			last = len(lines) - 1
			lines[last] = strings.ReplaceAll(lines[last], ",", "")

			var form Form
			var s = strings.Join(lines, "\n")
			if err := json.Unmarshal([]byte(s), &form); err != nil {
				return nil, errors.Wrap(err, "unable unmarshal json")
			}

			claim := &model.Claim{
				Code:       form.ID,
				Created:    form.Created.Time,
				DistrictID: Districts[form.FormID].ID,
				District:   Districts[form.FormID].Title,
			}

			for _, f := range form.Data {

				value := f.Value[0]

				//fmt.Println(form.FormID)

				switch Forms[form.FormID][f.FID] {
				case formstruct.StateKind:
					claim.Company.Activity = value
				case formstruct.StateName:
					claim.Company.Title = value
				case formstruct.StateAddress:
					claim.Company.Address = value
				case formstruct.StateINN:
					re := regexp.MustCompile(`\D`)
					claim.Company.INN = re.ReplaceAllString(value, "")
				case formstruct.StateFIO:
					fio := strings.Split(value, " ")

					if len(fio) < 3 {
						claim.Valid = false
						reason := "Нет данных по ФИО руководителя"
						claim.Reason = &reason
					} else {
						claim.Company.Head = model.Person{
							FIO: model.FIO{
								Surname:    fio[0],
								Name:       fio[1],
								Patronymic: fio[2],
							},
						}
					}
				case formstruct.StatePhone:
					claim.Company.Head.Contact.Phone = value
				case formstruct.StateEMail:
					claim.Company.Head.Contact.EMail = value
				case formstruct.StateCars:
					claim.Source = value
					claim.Cars = formstruct.ParseCars(value)
				case formstruct.StateAgreement:
					claim.Agreement = line
				case formstruct.StateReliability:
					claim.Reliability = line
				}
			}

			claims = append(claims, claim)
			continue
		}

		if inClaim == false {
			continue
		}

		m := object.FindAllStringSubmatch(line, -1)
		if len(m) > 0 {
			line = fmt.Sprintf("\t\"_id\" : \"%s\",", m[0][1])
		}

		lines = append(lines, line)
	}

	return claims, nil
}

func (p *Parser) printJSON(claim *model.Claim) {

	data, _ := json.MarshalIndent(claim, "", "\t")
	fmt.Printf("%s\n", string(data))

}
