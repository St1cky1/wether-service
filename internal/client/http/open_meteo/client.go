// Package openmeteo предоставляет клиент для работы с API Open-Meteo.
// Позволяет получать данные о погоде по географическим координатам.
package openmeteo

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Response struct {
	Current struct {
		Time          string  `json:"time"`
		Temperature2m float64 `json:"temperature_2m"`
	}
}

type client struct { // кастомный клиент
	httpClient *http.Client
}

func NewClient(httpClient *http.Client) *client { // конструктор клиента
	return &client{
		httpClient: httpClient,
	}
}

func (c *client) GetTemperature(lat, longt float64) (Response, error) { // метод ждя получения температуры по координатам

	resp, err := c.httpClient.Get(
		fmt.Sprintf("https://api.open-meteo.com/v1/forecast?latitude=%f&longitude=%f&current=temperature_2m", lat, longt),
	)
	if err != nil {
		return Response{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Response{}, fmt.Errorf("status code:%d", resp.StatusCode)

	}

	var response Response
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return Response{}, err
	}

	return response, nil
}
