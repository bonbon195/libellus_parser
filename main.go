package main

import (
	"bytes"
	"encoding/json"
	"libellus_parser/driveparser"
	"libellus_parser/helper"
	"libellus_parser/model"
	"libellus_parser/sheetparser"
	"libellus_parser/siteparser"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"google.golang.org/api/drive/v3"
)

var studentSchedule *[]model.Faculty
var teacherSchedule []model.Teacher
var teacherConsultations map[string][]model.ConsultTeacher

var mutex *sync.Mutex
var w *sync.WaitGroup
var service *drive.Service

var serverUrl string
var token string
var downloadsPath = "downloads/*.xlsx"

func main() {
	studentSchedule = &siteparser.Faculties

	mutex = &sync.Mutex{}
	w = &sync.WaitGroup{}

	err := godotenv.Load()
	if err != nil {
		log.Println(err)
	}
	serverUrl = os.Getenv("SERVER_URL")
	token = os.Getenv("TOKEN")
	for {
		err := updateDb()
		if err != nil {
			log.Println(err)
		}

		time.Sleep(time.Minute * 10)
	}

}

func updateDb() error {
	err := getFileIds()
	if err != nil {
		*studentSchedule = nil
		return err
	}
	client, err := driveparser.GetClient()
	if err != nil {
		return err
	}
	service, err = driveparser.GetService(client)
	if err != nil {
		return err
	}
	err = getFiles()
	if err != nil {
		return err
	}
	err = prepareData()
	if err != nil {
		*studentSchedule = nil
		teacherSchedule = nil
		teacherConsultations = nil

		return err
	}
	err = postData()
	if err != nil {
		log.Println(err)
	}

	deleteFiles()

	siteparser.ConsultationsId = ""
	*studentSchedule = nil
	teacherSchedule = nil
	teacherConsultations = nil

	return nil
}

func getFileIds() error {
	w.Add(1)
	doc, err := siteparser.GetWebPage()
	if err != nil {
		w.Done()
		return err
	}
	siteparser.ParseWebPage(doc)
	log.Println(*studentSchedule)
	w.Done()
	return nil
}

func postData() error {
	var body struct {
		StudentSchedule      []model.Faculty                   `json:"student_schedule"`
		TeacherSchedule      []model.Teacher                   `json:"teacher_schedule"`
		TeacherConsultations map[string][]model.ConsultTeacher `json:"teacher_consultations"`
		UpdateDate           string                            `json:"update_date"`
	}
	body.StudentSchedule = *studentSchedule
	body.TeacherSchedule = teacherSchedule
	body.TeacherConsultations = teacherConsultations

	loc := time.FixedZone("UTC+5", 5*60*60)
	t := time.Now().UTC().In(loc)

	body.UpdateDate = t.Format(time.DateTime)

	b, err := json.Marshal(&body)
	if err != nil {
		return err
	}

	reader := bytes.NewReader(b)
	req, err := http.NewRequest("POST", serverUrl+"/setData", reader)
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Token", token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	log.Println(res.Status)
	return nil

}

func getFiles() error {
	var err error
	for i, v := range *studentSchedule {
		w.Add(1)
		go func(id string, i int, code string) {
			//err = driveparser.GetFile(service, id, "downloads/"+strconv.Itoa(i)+"-"+code)
			err = driveparser.GetFile(service, id, "downloads/"+strconv.Itoa(i))
			w.Done()
		}(v.Id, i, v.Code)
	}

	w.Add(1)
	go func(id string, fileName string) {
		err = driveparser.GetFile(service, id, fileName)
		w.Done()
	}(siteparser.ConsultationsId, "consult")
	w.Wait()
	return err

}

func prepareTeachersData() {
	var schedule = make(map[string]map[string][]model.TeacherLesson)
	for _, faculty := range *studentSchedule {
		for _, group := range faculty.Groups {
			for _, day := range group.Days {
				for _, lesson := range day.Lessons {
					if !helper.IsNotEmpty(lesson.Teacher) {
						continue
					}
					replacedTeacherName := strings.ReplaceAll(lesson.Teacher, "\n", "")
					replacedDate := strings.ReplaceAll(strings.ReplaceAll(day.Date, "\n", ""), " ", "")
					if _, teacherExists := schedule[replacedTeacherName]; !teacherExists {
						schedule[replacedTeacherName] = make(map[string][]model.TeacherLesson)
					}
					if _, dateExists := schedule[replacedTeacherName]; !dateExists {
						schedule[replacedTeacherName][replacedDate] = make([]model.TeacherLesson, 0)
					}
					teacherLesson := model.TeacherLesson{Name: lesson.Name, Group: group.Name, Classroom: lesson.Classroom, Subgroup: lesson.Subgroup, Time: lesson.Time, Height: lesson.Height}
					schedule[replacedTeacherName][replacedDate] = append(schedule[replacedTeacherName][replacedDate], teacherLesson)
				}
			}
		}
	}

	for teacherName, weekMap := range schedule {
		var week = make([]model.TeacherDay, 0)
		for date, lessons := range weekMap {
			week = append(week, model.TeacherDay{Date: date, Lessons: lessons})
		}
		teacherSchedule = append(teacherSchedule, model.Teacher{Name: teacherName, Week: week})
	}
}

/*
	func prepareOldData() {
		w.Add(1)
		go func() {
			for _, v := range *studentSchedule {
				code := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(v.Code, "\n", ""), ".", "_"), " ", "")
				scheduleData[code] = make(map[string]map[string]model.Day)
				for _, group := range v.Groups {
					groupName := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(group.Name, "\n", ""), ".", "_"), " ", "")
					scheduleData[code][groupName] = make(map[string]model.Day)
					for _, day := range group.Days {
						date := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(group.Name, "\n", ""), ".", "_"), " ", "")
						scheduleData[code][groupName][date] = day
					}
				}
			}
			w.Done()
		}()
		w.Add(1)
		go func() {
			for k, v := range teacherConsultations {
				consultationsData[k] = make(map[string][]model.ConsultDay)
				for _, teacher := range v {
					name := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(teacher.Name, "\n", ""), " ", "-"), ".", "_")
					consultationsData[k][name] = teacher.Week
				}
			}
			w.Done()
		}()
	}
*/
func prepareData() error {
	var err error
	files, _ := filepath.Glob(downloadsPath)
	sort.Slice(files, func(i, j int) bool {
		a, _ := strings.CutPrefix(files[i], "downloads\\")
		b, _ := strings.CutPrefix(files[j], "downloads\\")
		a = strings.Split(a, ".")[0]
		b = strings.Split(b, ".")[0]
		aConv, _ := strconv.ParseInt(a, 10, 64)
		bConv, _ := strconv.ParseInt(b, 10, 64)
		return aConv < bConv
	})
	for i, v := range files {
		w.Add(1)
		go func(v string, index int) {
			var groups []model.Group
			groups, err = sheetparser.ParseScheduleSheet(v)
			mutex.Lock()
			(*studentSchedule)[index].Groups = groups
			mutex.Unlock()
			w.Done()
		}(v, i)
	}
	w.Wait()
	w.Add(1)
	go func() {
		prepareTeachersData()
		w.Done()
	}()
	w.Add(1)
	go func() {
		defer func() {
			if err := recover(); err != nil {
				w.Done()
			}
		}()
		teacherConsultations, err = sheetparser.ParseConsultationsFile("consult.xlsx")
		w.Done()
	}()
	w.Wait()
	return err
}

func deleteFiles() {
	files, _ := filepath.Glob(downloadsPath)
	for _, file := range files {
		w.Add(1)
		go func(file string) {
			err := os.Remove(file)
			if err != nil {
				log.Println(err)
			}
			w.Done()
		}(file)
	}
	w.Add(1)
	go func() {
		err := os.Remove("consult.xlsx")
		if err != nil {
			log.Println(err)
		}
		w.Done()
	}()
	w.Wait()
	return
}
