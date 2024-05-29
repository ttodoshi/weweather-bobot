package wttrin

import (
	"fmt"
	"io"
	"net/http"

	"github.com/ttodoshi/weweather-bobot/internal/core/ports"
)

type wttrInWeatherProvider struct{}

func NewWttrInWeatherProvider() ports.WeatherProvider {
	return &wttrInWeatherProvider{}
}

func (s *wttrInWeatherProvider) FetchWeather(cities []string) (response []string) {
	// Получение погоды с сайта https://wttr.in/
	client := &http.Client{}

	for _, city := range cities {
		url := `https://wttr.in/` + city + `?format=4`
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			fmt.Println(err)
			return
		}
		req.Header.Add("Accept-Language", "ru")

		resp, err := client.Do(req)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Println(err)
			return
		}
		response = append(response, string(body))
	}

	return
}
