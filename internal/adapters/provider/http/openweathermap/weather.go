package openweathermap

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/ttodoshi/weweather-bobot/internal/core/ports"
)

type openWeatherMapWeatherProvider struct {
	apiKey string
}

func NewOpenWeatherMapWeatherProvider(apiKey string) ports.WeatherProvider {
	return &openWeatherMapWeatherProvider{
		apiKey: apiKey,
	}
}

func (s *openWeatherMapWeatherProvider) FetchWeather(cities []string) (response []string) {
	// Получение погоды с сайта https://openweathermap.org/
	client := &http.Client{}

	for _, city := range cities {
		locationUrl := fmt.Sprintf(
			"http://api.openweathermap.org/geo/1.0/direct?q=%s&appid=%s&limit=1",
			city,
			s.apiKey,
		)
		req, err := http.NewRequest("GET", locationUrl, nil)
		if err != nil {
			fmt.Println(err)
			return
		}

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
		var cityLocations []map[string]any
		err = json.Unmarshal(body, &cityLocations)
		if err != nil {
			fmt.Println(err)
			return
		}
		if len(cityLocations) == 0 {
			response = append(
				response,
				fmt.Sprintf("Город %s не найден", city),
			)
			continue
		}
		cityLocation := cityLocations[0]

		url := fmt.Sprintf(
			"https://api.openweathermap.org/data/2.5/weather?lat=%f&lon=%f&appid=%s&units=metric&lang=ru",
			cityLocation["lat"].(float64),
			cityLocation["lon"].(float64),
			s.apiKey,
		)
		req, err = http.NewRequest("GET", url, nil)
		if err != nil {
			fmt.Println(err)
			return
		}

		resp, err = client.Do(req)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer resp.Body.Close()

		body, err = io.ReadAll(resp.Body)
		if err != nil {
			fmt.Println(err)
			return
		}
		var res map[string]any
		err = json.Unmarshal(body, &res)
		if err != nil {
			fmt.Println(err)
			return
		}

		response = append(
			response,
			fmt.Sprintf(
				"%s: %s 🌡️%.1f⁰C 🌬️ %.1f km/h\n",
				city,
				res["weather"].([]any)[0].(map[string]any)["description"].(string),
				res["main"].(map[string]any)["temp"].(float64),
				res["wind"].(map[string]any)["speed"].(float64),
			),
		)
	}

	return
}
