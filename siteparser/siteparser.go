package siteparser

import (
	"io"
	"libellus_parser/helper"
	"libellus_parser/model"
	"log"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const urtkUrl string = "https://urtt.ru/students/dnevnoe/raspisaniya/"

var Specialities []model.Speciality
var ConsultationsId string

func GetWebPage() (*goquery.Document, error) {
	res, err := http.Get(urtkUrl)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			return
		}
	}(res.Body)
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, err
	}
	return doc, nil
}

func ParseWebPage(doc *goquery.Document) {
	doc.Find(".gem-list").Each(func(i int, selection *goquery.Selection) {
		selection.Find("li").Each(func(i2 int, selection2 *goquery.Selection) {
			Specialities = append(Specialities, model.Speciality{
				Code: selection2.Text()[:8],
				Id:   strings.Split(helper.First(selection2.Find("a").Attr("href")), "/")[5]})
		})
	})
	doc.Find("p").Each(func(i int, s *goquery.Selection) {
		s.Find("strong").Each(func(i int, s2 *goquery.Selection) {
			if strings.Contains(s2.Text(), "кон­суль­та­ций") {
				s.Find("a").Each(func(i int, s3 *goquery.Selection) {
					ConsultationsId = strings.Split(helper.First(s3.Attr("href")), "/")[5]
					log.Println(ConsultationsId)
				})
			}
		})
	})
}
