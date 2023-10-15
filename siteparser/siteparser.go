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

var Faculties []model.Faculty
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

// type urls struct{
// 	first string
// 	last string
// }

func ParseWebPage(doc *goquery.Document) {
	codes := make([]string, 0)
	ids := make([]string, 0)
	doc.Find(".gem-list-type-star").Each(func(i int, selection *goquery.Selection) {
		selection.Find("ul").Each(func(i2 int, selection2 *goquery.Selection) {
			selection2.Find("li").Each(func(i int, s *goquery.Selection) {
				codes = append(codes, strings.Split(selection2.Text(), " ")[0])

			})
		})
		selection.Find("p").Each(func(i2 int, selection2 *goquery.Selection) {
			selection3 := selection2.Find("a")
			// hrefFirst, _ := selection3.First().Attr("href")
			hrefLast, _ := selection3.Last().Attr("href")
			ids = append(ids, strings.Split(hrefLast, "/")[5])
			// specUrls = append(specUrls, urls{strings.Split(hrefFirst, "/")[5], strings.Split(hrefLast, "/")[5]})
		})
	})
	for i, v := range codes {
		Faculties = append(Faculties, model.Faculty{
			Code: v,
			Id:   ids[i]})
	}
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
