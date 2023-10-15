package model

type ConsultTeacher struct {
	Name string       `json:"name"`
	Week []ConsultDay `json:"week"`
}
