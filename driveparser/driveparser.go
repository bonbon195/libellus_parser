package driveparser

import (
	"io"
	"net/http"
	"os"

	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

func GetClient() (*http.Client, error) {
	config, err := google.JWTConfigFromJSON([]byte(os.Getenv("SERVICE_ACCOUNT_CREDENTIALS")), drive.DriveReadonlyScope)
	if err != nil {
		return nil, err
	}
	return config.Client(context.Background()), nil
}

func GetService(client *http.Client) (*drive.Service, error) {
	service, err := drive.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}
	return service, nil
}

func GetFile(service *drive.Service, fileId string, fileName string) error {
	r, err := service.Files.Export(fileId,
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet").Download()
	if err != nil {
		return err
	}
	file, err := os.Create(fileName + ".xlsx")
	if err != nil {
		return err
	}
	result, _ := io.ReadAll(r.Body)
	var err1 error = nil
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			err1 = err
		}
	}(r.Body)
	_, err = file.Write(result)
	if err != nil {
		return err
	}

	err = file.Close()
	if err != nil {
		return err
	}
	return err1
}
