package main

import (
	"libellus_parser/driveparser"
	"libellus_parser/firebasesdk"
	"libellus_parser/model"
	"libellus_parser/sheetparser"
	"libellus_parser/siteparser"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"golang.org/x/exp/maps"
	"google.golang.org/api/drive/v2"
)

var scheduleData map[string]map[string]map[string]model.Day

var consultationsData map[string]map[string][]model.ConsultDay
var mutex *sync.Mutex
var w *sync.WaitGroup
var specialities *[]model.Speciality
var service *drive.Service

func main() {
	specialities = &siteparser.Specialities

	scheduleData = make(map[string]map[string]map[string]model.Day)
	consultationsData = make(map[string]map[string][]model.ConsultDay)
	mutex = &sync.Mutex{}
	w = &sync.WaitGroup{}

	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}

	for {
		err := updateDb()
		if err != nil {
			log.Println(err)
		}

		time.Sleep(time.Minute * 10)
	}

}

func getFileIds() error {
	w.Add(1)
	doc, err := siteparser.GetWebPage()
	if err != nil {
		w.Done()
		return err
	}
	siteparser.ParseWebPage(doc)
	log.Println(*specialities)
	w.Done()
	return nil
}

func updateDb() error {
	client, err := driveparser.GetClient()
	if err != nil {
		return err
	}
	service, err = driveparser.GetService(client)
	if err != nil {
		return err
	}
	err = getFileIds()
	if err != nil {
		*specialities = nil
		return err
	}
	err = getFiles()
	if err != nil {
		return err
	}
	err = prepareData()
	if err != nil {
		maps.Clear(scheduleData)
		maps.Clear(consultationsData)
		return err
	}
	err = firebasesdk.SendData(&scheduleData, &consultationsData)
	if err != nil {
		return err
	}
	err = deleteFiles()
	if err != nil {
		log.Println(err)
	}
	maps.Clear(scheduleData)
	maps.Clear(consultationsData)
	siteparser.ConsultationsId = ""
	*specialities = nil
	return nil
}

func getFiles() error {
	var err error
	for _, v := range *specialities {
		w.Add(1)
		go func(id string, code string) {
			err = driveparser.GetFile(service, id, code)
			w.Done()
		}(v.Id, v.Code)
	}

	w.Add(1)
	go func(id string, fileName string) {
		err = driveparser.GetFile(service, id, fileName)
		w.Done()
	}(siteparser.ConsultationsId, "consult")
	w.Wait()
	return err

}

func prepareData() error {
	var err error
	for _, v := range *specialities {
		w.Add(1)
		go func(code string) {
			var groups map[string]map[string]model.Day
			groups, err = sheetparser.ParseScheduleSheet(code + ".xlsx")
			mutex.Lock()
			scheduleData[strings.ReplaceAll(code, ".", "_")] = groups
			mutex.Unlock()
			w.Done()
		}(v.Code)
	}
	w.Add(1)
	go func() {
		defer func() {
			if err := recover(); err != nil {
				log.Println("panik", err)
				w.Done()
			}
		}()
		consultationsData, err = sheetparser.ParseConsultationsFile("consult.xlsx")
		w.Done()
	}()
	w.Wait()
	return err
}

func deleteFiles() error {
	for _, v := range *specialities {
		err := os.Remove(v.Code + ".xlsx")
		if err != nil {
			log.Println(err)
		}
	}
	err := os.Remove("consult.xlsx")
	if err != nil {
		log.Println(err)
	}
	return nil
}
