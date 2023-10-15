package sheetparser

import (
	"github.com/xuri/excelize/v2"
	"libellus_parser/helper"
	"libellus_parser/model"
	"log"
	"strings"
)

func ParseScheduleSheet(fileName string) ([]model.Group, error) {

	f, err := excelize.OpenFile(fileName)
	if err != nil {
		return nil, err
	}

	sheets := f.WorkBook.Sheets.Sheet
	if err != nil {
		return nil, err
	}
	var groups = make([]model.Group, 0)
	for _, sheet := range sheets {
		name := &sheet.Name
		cols, err := f.GetCols(*name)
		if err != nil {
			return nil, err
		}
		sheetGroups, err := GetGroups(&cols, f, name)
		if err != nil {
			return nil, err
		}
		groups = append(sheetGroups, sheetGroups...)
	}
	defer func(f *excelize.File) {
		err := f.Close()
		if err != nil {
			log.Println(err)
		}
	}(f)

	return groups, nil
}

func GetGroups(cols *[][]string, f *excelize.File, name *string) ([]model.Group, error) {
	rows, err := f.GetRows(*name)
	if err != nil {
		return nil, err
	}
	//groups := make(map[string]map[string]model.Day)
	groups := make([]model.Group, 0)

	groupsRow := 0
	groupsCol := 0
	for i, col := range *cols {
		for j, row := range col {
			if strings.Contains(row, "Группа") {
				groupsCol = i
				groupsRow = j
			}
		}
	}
	size := getDaySize(cols, groupsRow+2)
	for i, v := range rows[groupsRow][groupsCol+2:] {
		if helper.IsNotEmpty(v) {
			days, err := getDays(cols, f, name, i+groupsCol+2, groupsRow+2, size)
			if err != nil {
				return nil, err
			}
			groups = append(groups, model.Group{Name: v, Days: days})
		}

	}
	return groups, nil
}

func getDays(cols *[][]string, f *excelize.File, name *string, colNum int, dayStartPos int, size int) ([]model.Day, error) {
	var days = make([]model.Day, 0)
	for i, v := range (*cols)[0][dayStartPos:] {
		if helper.IsNotEmpty(v) {
			s := strings.Split(v, " ")
			date := s[0]
			lessons, err := getLessons(cols, f, name, &colNum, i+dayStartPos, size)
			if err != nil {
				return nil, err
			}
			days = append(days, model.Day{Name: s[2], Date: date, Lessons: lessons})
		}
	}
	return days, nil
}

func getDaySize(cols *[][]string, dayStartPos int) int {
	daySize := 0
	count := 0
	for i, s := range (*cols)[0][dayStartPos:] {
		if helper.IsNotEmpty(s) {
			count++
			if count == 2 {
				daySize = i
				break
			}
		}
	}
	return daySize
}

func getLessons(cols *[][]string, f *excelize.File, name *string, colNum *int, dayBlock int, size int) ([]model.Lesson, error) {
	var lessons []model.Lesson
	mergedLessons, err := getMergedLessons(f, name)
	if err != nil {
		return nil, err
	}

	for i := 0; i < size; i += 2 {
		name1 := (*cols)[*colNum][dayBlock+i]
		classroom1 := (*cols)[*colNum+1][dayBlock+i]
		teacher1 := (*cols)[*colNum][dayBlock+i+1]
		name2 := (*cols)[*colNum+2][dayBlock+i]
		classroom2 := (*cols)[*colNum+3][dayBlock+i]
		teacher2 := (*cols)[*colNum+2][dayBlock+i+1]
		time := (*cols)[3][dayBlock+i]

		var lesson model.Lesson

		if val, ok := mergedLessons[cellCoords{*colNum + 1, dayBlock + i + 1}]; ok { // учимся вместе весь день
			lesson = model.Lesson{Name: val.value, Teacher: "", Classroom: "", Subgroup: 0, Time: time, Height: val.height}

		} else if helper.IsNotEmpty(name1) && helper.IsNotEmpty(classroom1) && helper.IsNotEmpty(teacher1) && // учится 1 группа
			!helper.IsNotEmpty(name2) && !helper.IsNotEmpty(classroom2) && !helper.IsNotEmpty(teacher2) {

			lesson = model.Lesson{Name: name1, Teacher: teacher1, Classroom: classroom1, Subgroup: 1, Time: time, Height: 1}

		} else if !helper.IsNotEmpty(name1) && !helper.IsNotEmpty(classroom1) && !helper.IsNotEmpty(teacher1) && // учится 2 группа
			helper.IsNotEmpty(name2) && helper.IsNotEmpty(classroom2) && helper.IsNotEmpty(teacher2) {

			lesson = model.Lesson{Name: name2, Teacher: teacher2, Classroom: classroom2, Subgroup: 2, Time: time, Height: 1}

		} else if helper.IsNotEmpty(name1) && helper.IsNotEmpty(classroom1) && helper.IsNotEmpty(teacher1) && // учатся обе группы
			helper.IsNotEmpty(name2) && helper.IsNotEmpty(classroom2) && helper.IsNotEmpty(teacher2) {

			lessons = append(lessons, model.Lesson{Name: name1, Teacher: teacher1, Classroom: classroom1, Subgroup: 1, Time: time, Height: 1})
			lesson = model.Lesson{Name: name2, Teacher: teacher2, Classroom: classroom2, Subgroup: 2, Time: time, Height: 1}

		} else if helper.IsNotEmpty(name1) && !helper.IsNotEmpty(classroom1) && helper.IsNotEmpty(teacher1) && // учимся вместе
			!helper.IsNotEmpty(name2) && helper.IsNotEmpty(classroom2) && !helper.IsNotEmpty(teacher2) {

			lesson = model.Lesson{Name: name1, Teacher: teacher1, Classroom: classroom2, Subgroup: 0, Time: time, Height: 1}

		} else if !helper.IsNotEmpty(name1) && !helper.IsNotEmpty(classroom1) && !helper.IsNotEmpty(teacher1) && // нет пар
			!helper.IsNotEmpty(name2) && !helper.IsNotEmpty(classroom2) && !helper.IsNotEmpty(teacher2) {

			lesson = model.Lesson{Name: "", Teacher: "", Classroom: "", Subgroup: 0, Time: time, Height: 1}
		}
		lessons = append(lessons, lesson)
	}

	return lessons, nil
}

type mergeCell struct {
	height int
	value  string
}

type cellCoords struct {
	x int
	y int
}

func getMergedLessons(f *excelize.File, name *string) (map[cellCoords]mergeCell, error) {
	merged, err := f.GetMergeCells(*name)
	if err != nil {
		return nil, err
	}
	mergedLessons := make(map[cellCoords]mergeCell)
	for _, v := range merged {
		var coords []int
		for _, c := range strings.Split(v[0], ":") {
			x, y, err := excelize.CellNameToCoordinates(c)
			if err != nil {
				return nil, err
			}
			coords = append(coords, x, y)
		}
		height := coords[3] - coords[1]
		if height > 2 && !strings.Contains(v[1], "ПОНЕДЕЛЬНИК") && !strings.Contains(v[1], "ВТОРНИК") && !strings.Contains(v[1], "СРЕДА") &&
			!strings.Contains(v[1], "ЧЕТВЕРГ") && !strings.Contains(v[1], "ПЯТНИЦА") && !strings.Contains(v[1], "СУББОТА") && !strings.Contains(v[1], "ВОСКРЕСЕНЬЕ") {
			mergedLessons[cellCoords{x: coords[0], y: coords[1]}] = mergeCell{height: (height + 1) / 2, value: v[1]}
		}

	}
	return mergedLessons, nil
}
