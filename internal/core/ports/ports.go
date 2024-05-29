package ports

// Интерфейс для получения погоды
type WeatherProvider interface {
	FetchWeather(cities []string) []string
}
