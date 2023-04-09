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

/* func ParseConsultationsFile(fileName string) (map[string][]model.ConsultDay, error) {
	f, err := excelize.OpenFile(fileName)
	if err != nil {
		return nil, err
	}
	sheets := f.WorkBook.Sheets.Sheet
	if err != nil {
		return nil, err
	}
	var name = ""
	today := time.Now().In(time.FixedZone("+05", 18000))
	for _, sheet := range sheets {
		if sheet.State == "visible" {
			name = sheet.Name
			go func (name string)  {

			}(name)
			dates := strings.Split(name, " ")
			var date1 time.Time = time.Time{}
			var date2 time.Time = time.Time{}
			for i, dateStr := range dates {
				if strings.Contains(dateStr, ".") {
					if i == 0 {
						date1, err = parseDate(dateStr, true)
						if err != nil {
							return nil, err
						}
					} else {
						date2, err = parseDate(dateStr, false)
						if err != nil {
							return nil, err
						}
					}
				}

			}
			if today.Before(date1) && today.Before(date2) {
				break
			} else if today.After(date1) && today.Before(date2) {
				break
			}
		}
	}
	log.Println(name)
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
} */

func ParseConsultationsFile(fileName string) (map[string]map[string][]model.ConsultDay, error) {

	f, err := excelize.OpenFile(fileName)
	if err != nil {
		return nil, err
	}
	sheets := f.WorkBook.Sheets.Sheet
	if err != nil {
		return nil, err
	}

	var name string
	teachers := make(map[string]map[string][]model.ConsultDay)
	mutex := &sync.Mutex{}
	w := &sync.WaitGroup{}
	for _, sheet := range sheets {
		if sheet.State == "visible" {
			name = sheet.Name
			w.Add(1)
			go func(name string, file *excelize.File) {
				var teachersFromSheet map[string][]model.ConsultDay
				teachersFromSheet, err = parseSheet(name, file)
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
				teachers[name] = teachersFromSheet
				mutex.Unlock()
				w.Done()
			}(name, f)
		}
	}
	w.Wait()
	return teachers, err
}

func parseSheet(name string, f *excelize.File) (map[string][]model.ConsultDay, error) {
	log.Println(name)
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

/* func parseDate(str string, isFirst bool) (time.Time, error) {
	var month = 1
	var day = 1
	re := regexp.MustCompile(`([\d]{2})\.([\d]{2})`)
	dates := re.FindAllString(str, 2)
	for _, v := range dates {
		splittedStr := strings.Split(v, ".")
		for i, v := range splittedStr {
			if i == 0 {
				dayy, err := strconv.Atoi(v)
				if err != nil {
					return time.Time{}, err
				}
				day = dayy
			} else {
				monthh, err := strconv.Atoi(v)

				if err != nil {
					return time.Time{}, err
				}
				month = monthh
			}
		}
	}

	hour := 0
	min := 0
	if !isFirst {
		hour = 23
		min = 59
	}
	date := time.Date(time.Now().Year(), time.Month(month), day, hour, min, min, min, time.FixedZone("+05", +18000))
	return date, nil
} */

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

func getTeachers(startCol *int, startRow *int, cols *[][]string, sheetName *string, f *excelize.File) (map[string][]model.ConsultDay, error) {
	teachers := make(map[string][]model.ConsultDay)
	teachersCol := (*cols)[*startCol][*startRow+2:]
	height := 0
	var previousTeacher string
	for i := 0; i < len(teachersCol); i++ {
		height++
		reg, err := regexp.Compile(`(^[^A-Za-zА-Яа-яёЁ\-$]+|[^A-Za-zА-Яа-яёЁ\-$]+\n){1}`)
		if err != nil {
			return nil, err
		}
		teacherSelector := reg.ReplaceAllLiteralString(teachersCol[i], "")
		teacherSelector = strings.Replace(teacherSelector, " ", "_", -1)

		if helper.IsNotEmpty(teacherSelector) || (i+1) == len(teachersCol) {
			if previousTeacher != "" {

				teachers[previousTeacher] = getConsultDays(startCol, startRow, cols, *startRow+i+2-height, sheetName, f)
				height = 0
			}
			previousTeacher = teacherSelector
		}

	}
	return teachers, nil
}

func getConsultDays(startCol *int, startRow *int, cols *[][]string, teacherRow int, sheetName *string, f *excelize.File) []model.ConsultDay {
	weekCols := (*cols)[*startCol+2:]
	dayMap := getConsultDayColumns(&weekCols, startCol, startRow)
	var days []model.ConsultDay
	for el := dayMap.Front(); el != nil; el = el.Next() {
		day := getSingleConsultDay(&el.Key, &el.Value[0], &el.Value[1], cols, &teacherRow, sheetName, f)
		days = append(days, day)
	}
	return days
}

func getConsultDayColumns(weekCols *[][]string, startCol *int, startRow *int) *orderedmap.OrderedMap[string, [2]int] {
	days := orderedmap.NewOrderedMap[string, [2]int]()
	for i := 0; i < len(*weekCols); i++ {
		v := (*weekCols)[i][*startRow]
		if strings.Contains(strings.ToLower(v), "понедельник") {
			days.Set("Понедельник", [2]int{i + 2 + *startCol, *startRow})
		}
		if strings.Contains(strings.ToLower(v), "вторник") {
			days.Set("Вторник", [2]int{i + 2 + *startCol, *startRow})
		}
		if strings.Contains(strings.ToLower(v), "среда") {
			days.Set("Среда", [2]int{i + 2 + *startCol, *startRow})
		}
		if strings.Contains(strings.ToLower(v), "четверг") {
			days.Set("Четверг", [2]int{i + 2 + *startCol, *startRow})
		}
		if strings.Contains(strings.ToLower(v), "пятница") {
			days.Set("Пятница", [2]int{i + 2 + *startCol, *startRow})
		}
		if strings.Contains(strings.ToLower(v), "суббота") {
			days.Set("Суббота", [2]int{i + 2 + *startCol, *startRow})
		}
		if strings.Contains(strings.ToLower(v), "воскресенье") {
			days.Set("Воскресенье", [2]int{i + 2 + *startCol, *startRow})
		}

	}
	return days
}

func getSingleConsultDay(dayName *string, dayCol *int, dayRow *int, cols *[][]string, teacherRow *int, sheetName *string, f *excelize.File) model.ConsultDay {
	time := (*cols)[*dayCol][*teacherRow]
	classroom := (*cols)[*dayCol][*teacherRow+1]
	format := "dd-mm-yyyy"
	style, _ := f.NewStyle(&excelize.Style{CustomNumFmt: &format})
	cell, _ := excelize.CoordinatesToCellName(*dayCol+1, *dayRow+2)
	f.SetCellStyle(*sheetName, cell, cell, style)
	date, _ := f.GetCellValue(*sheetName, cell)
	return model.ConsultDay{Name: *dayName, Date: date, Time: time, Classroom: classroom}
}
