package model

type Faculty struct {
	Id     string  `json:"id"`
	Code   string  `json:"code"`
	Groups []Group `json:"groups"`
}
