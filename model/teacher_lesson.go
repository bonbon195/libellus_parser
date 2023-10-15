package model

type TeacherLesson struct {
	Name      string `json:"name"`
	Group     string `json:"group"`
	Classroom string `json:"classroom"`
	Time      string `json:"time"`
	Subgroup  int    `json:"subgroup"`
	Height    int    `json:"height"`
}
