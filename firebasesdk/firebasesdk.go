package firebasesdk

import (
	"context"
	"libellus_parser/model"
	"log"
	"os"

	firebase "firebase.google.com/go/v4"
	"github.com/joho/godotenv"
	"google.golang.org/api/option"
)

func SendData(scheduleData *map[string]map[string]map[string]model.Day, consultationsData *map[string]map[string][]model.ConsultDay) error {

	err := godotenv.Load()
	if err != nil {
		return err
	}

	ctx := context.Background()

	conf := &firebase.Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
	}

	opt := option.WithCredentialsFile(os.Getenv("CREDENTIALS_FILE"))
	app, err := firebase.NewApp(context.Background(), conf, opt)
	if err != nil {
		return err
	}

	client, err := app.Database(ctx)
	if err != nil {
		return err
	}

	ref := client.NewRef("specialities")
	if err := ref.Set(ctx, scheduleData); err != nil {
		return err
	}

	ref2 := client.NewRef("consultations")
	if err := ref2.Set(ctx, consultationsData); err != nil {
		return err
	}

	log.Println("Done! All data has been successfully put to db!")
	return nil
}
