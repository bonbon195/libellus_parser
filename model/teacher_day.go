package model

type TeacherDay struct {
	Date    string          `json:"date"`
	Lessons []TeacherLesson `json:"lessons"`
}
