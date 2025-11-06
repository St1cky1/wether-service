// Package geocoding предоставляет клиент для работы с API geo-coding.
// Позволяет получать координаты города по его названию.
package geocoding

// API для получения координат города по названию

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Responce struct { // структура ответа
	Name      string  `json:"name"`
	Country   string  `json:"country"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type client struct { // структура клиента
	httpClient *http.Client
}

func NewClient(httpClient *http.Client) *client { // конструктор клиента
	return &client{
		httpClient: httpClient,
	}
}

func (c *client) GetCoords(city string) (Responce, error) { // метод для получения координат

	resp, err := c.httpClient.Get(
		fmt.Sprintf("https://geocoding-api.open-meteo.com/v1/search?name=%s&count=1&languahe=ru&format=json", city),
	)
	if err != nil {
		return Responce{}, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Responce{}, fmt.Errorf("status code: %d", resp.StatusCode)
	}

	var geoResp struct {
		Result []Responce `json:"results"`
	}

	err = json.NewDecoder(resp.Body).Decode(&geoResp)
	if err != nil {
		return Responce{}, err
	}

	return geoResp.Result[0], nil

}
