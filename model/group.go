package model

type Group struct {
	Name string `json:"name"`
	Days []Day  `json:"days"`
}
