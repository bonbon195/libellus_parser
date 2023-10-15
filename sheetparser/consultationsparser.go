package sheetparser

import (
	"errors"
	"libellus_parser/helper"
	"libellus_parser/model"
	"log"
	"regexp"
	"strings"
	"sync"

	"github.com/elliotchance/orderedmap/v2"
	"github.com/xuri/excelize/v2"
)

func ParseConsultationsFile(fileName string) (map[string][]model.ConsultTeacher, error) {

	f, err := excelize.OpenFile(fileName)
	if err != nil {
		return nil, err
	}
	sheets := f.WorkBook.Sheets.Sheet
	if err != nil {
		return nil, err
	}

	var name string
	teachers := make(map[string][]model.ConsultTeacher, 0)
	mutex := &sync.Mutex{}
	w := &sync.WaitGroup{}
	for _, sheet := range sheets {
		if sheet.State == "visible" {
			name = sheet.Name
			w.Add(1)
			go func(name string, file *excelize.File) {
				var sheetTeachers []model.ConsultTeacher
				sheetTeachers, err = parseSheet(name, file)
				if err != nil {
					w.Done()
					return
				}
				name, err = parseDate(name)
				if err != nil {
					w.Done()
					return
				}
				mutex.Lock()
				teachers[name] = sheetTeachers
				mutex.Unlock()
				w.Done()
			}(name, f)
		}
	}
	w.Wait()
	return teachers, err
}

func parseSheet(name string, f *excelize.File) ([]model.ConsultTeacher, error) {
	log.Println(name)
	defer func() {
		if r := recover(); r != nil {
			return
		}
	}()
	cols, err := f.GetCols(name)
	if err != nil {
		return nil, err
	}

	startCol, startRow := getStartCoords(&cols)
	teachers, err := getTeachers(&startCol, &startRow, &cols, &name, f)
	if err != nil {
		return nil, err
	}
	return teachers, nil
}
func parseDate(str string) (string, error) {
	re := regexp.MustCompile(`([\d]{2})\.([\d]{2})`)
	dates := re.FindAllString(str, 2)
	if len(dates) < 2 {
		return "", errors.New("wrong sheet name")
	}
	str = ""
	for i, v := range dates {
		str += strings.ReplaceAll(v, ".", "_")
		if i != len(dates)-1 {
			str += "-"
		}
	}
	return str, nil
}

func getStartCoords(cols *[][]string) (int, int) {
	for colNum, col := range *cols {
		for rowNum, v := range col {
			if strings.Contains(v, "ФИО преподавателя") {
				return colNum, rowNum
			}
		}
	}
	return 0, 0
}

func getTeachers(startCol *int, startRow *int, cols *[][]string, sheetName *string, f *excelize.File) ([]model.ConsultTeacher, error) {
	teachers := make([]model.ConsultTeacher, 0)
	teachersCol := (*cols)[*startCol][*startRow+2:]
	height := 0
	var previousTeacher string
	for i := 0; i < len(teachersCol); i++ {
		height++
		reg, err := regexp.Compile(`(^[^A-Za-zА-Яа-яёЁ-]+|[^A-Za-zА-Яа-яёЁ ]+|\n)`)
		if err != nil {
			return nil, err
		}
		regDoubleSpace, err := regexp.Compile(`[ ]{2,}`)
		if err != nil {
			return nil, err
		}
		teacherSelector := reg.ReplaceAllLiteralString(teachersCol[i], "")
		teacherSelector = regDoubleSpace.ReplaceAllLiteralString(teacherSelector, " ")
		teacherSelector = strings.TrimSpace(teacherSelector)
		if helper.IsNotEmpty(teacherSelector) || (i+1) == len(teachersCol) {
			if previousTeacher != "" {
				week, err := getConsultDays(startCol, startRow, cols, *startRow+i+2-height, sheetName, f)
				if err != nil {
					return nil, err
				}
				teachers = append(teachers, model.ConsultTeacher{Name: previousTeacher, Week: week})
				height = 0
			}
			previousTeacher = teacherSelector
		}

	}
	return teachers, nil
}

func getConsultDays(startCol *int, startRow *int, cols *[][]string, teacherRow int, sheetName *string, f *excelize.File) ([]model.ConsultDay, error) {
	weekCols := (*cols)[*startCol+2:]
	dayMap := getConsultDayColumns(&weekCols, startCol, startRow)
	var days []model.ConsultDay
	for el := dayMap.Front(); el != nil; el = el.Next() {
		day, err := getSingleConsultDay(&el.Key, &el.Value[0], &el.Value[1], cols, &teacherRow, sheetName, f)
		if err != nil {
			return nil, err
		}
		days = append(days, day)
	}
	return days, nil
}

func getConsultDayColumns(weekCols *[][]string, startCol *int, startRow *int) *orderedmap.OrderedMap[string, [2]int] {
	days := orderedmap.NewOrderedMap[string, [2]int]()
	for i := 0; i < len(*weekCols); i++ {
		v := strings.ToLower((*weekCols)[i][*startRow])
		if strings.Contains(v, "понедельник") {
			days.Set("Понедельник", [2]int{i + 2 + *startCol, *startRow})
		}
		if strings.Contains(v, "вторник") {
			days.Set("Вторник", [2]int{i + 2 + *startCol, *startRow})
		}
		if strings.Contains(v, "среда") {
			days.Set("Среда", [2]int{i + 2 + *startCol, *startRow})
		}
		if strings.Contains(v, "четверг") {
			days.Set("Четверг", [2]int{i + 2 + *startCol, *startRow})
		}
		if strings.Contains(v, "пятница") {
			days.Set("Пятница", [2]int{i + 2 + *startCol, *startRow})
		}
		if strings.Contains(v, "суббота") {
			days.Set("Суббота", [2]int{i + 2 + *startCol, *startRow})
		}
		if strings.Contains(v, "воскресенье") {
			days.Set("Воскресенье", [2]int{i + 2 + *startCol, *startRow})
		}

	}
	return days
}

func getSingleConsultDay(dayName *string, dayCol *int, dayRow *int, cols *[][]string, teacherRow *int, sheetName *string, f *excelize.File) (model.ConsultDay, error) {
	time := (*cols)[*dayCol][*teacherRow]
	classroom := (*cols)[*dayCol][*teacherRow+1]
	format := "dd-mm-yyyy"
	style, err := f.NewStyle(&excelize.Style{CustomNumFmt: &format})
	if err != nil {
		return model.ConsultDay{}, err
	}
	cell, err := excelize.CoordinatesToCellName(*dayCol+1, *dayRow+2)
	if err != nil {
		return model.ConsultDay{}, err
	}
	err = f.SetCellStyle(*sheetName, cell, cell, style)
	if err != nil {
		return model.ConsultDay{}, err
	}
	date, _ := f.GetCellValue(*sheetName, cell)
	return model.ConsultDay{Name: *dayName, Date: date, Time: time, Classroom: classroom}, nil
}
