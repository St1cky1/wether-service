package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/St1cky1/wether-service/internal/client/http/geocoding"
	openmeteo "github.com/St1cky1/wether-service/internal/client/http/open_meteo"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-co-op/gocron/v2"
	"github.com/jackc/pgx/v5"
	// "github.com/golang/protobuf/ptypes/timestamp"
)

const (
	portHTTP = ":3000"
	city     = "Moscow"
)

type Reading struct { // показания погоды
	name        string    `db:"name"`
	Timestamp   time.Time `db:"timestamp"`
	Temperature float64   `db:"temperature"`
}

func main() {

	r := chi.NewRouter()     //  создакм новый роутер, через него поднимаем сервер
	r.Use(middleware.Logger) // логируем запросы
	ctx := context.Background()

	// urlExample := "postgres://username:password@localhost:5432/database_name"
	conn, err := pgx.Connect(ctx, "postgresql://Vladi:password@localhost:54321/weather")
	if err != nil {
		panic(err)
	}
	defer conn.Close(ctx)

	r.Get("/{city}", func(w http.ResponseWriter, r *http.Request) {

		cityName := chi.URLParam(r, "city")          // получаем параметр из урла
		fmt.Printf("Requested city: %s\n", cityName) // лонируем запрос

		var reading Reading

		err := conn.QueryRow(ctx,
			"select name, timestamp, temperature from reading where name = $1 order by timestamp desc limit 1",
			city).Scan(&reading.name, &reading.Timestamp, &reading.Temperature)

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte("not found"))
				return
			}

			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("internal error"))
			return
		}

		var raw []byte
		raw, err = json.Marshal(reading)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("internal error"))
			return
		}

		w.Header().Set("Content-Type", "application/json") // укзываем, что данные вернем в формате json
		_, err = w.Write(raw)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("internal error"))
			return
		}
	})

	s, err := gocron.NewScheduler() // создаем новый экземпляр планировщика
	if err != nil {
		panic(err)
	}

	jobs, err := initJobs(ctx, s, conn) // вызываем "cron"
	if err != nil {
		panic(err)
	}

	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()

		fmt.Println("Starting HTTP server on port", portHTTP)

		if err := http.ListenAndServe(portHTTP, r); err != nil { // поднимаем сервер
			panic(err)
		}
	}()

	go func() {
		defer wg.Done()

		fmt.Printf("Starting job: %v\n", jobs[0].ID()) // проверяем, что все работает (возвращает уникальный идентификатор задания.)

		s.Start() // стартуем приложение cron
	}()

	wg.Wait()
}

func initJobs(ctx context.Context, sheduler gocron.Scheduler, conn *pgx.Conn) ([]gocron.Job, error) {

	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	geocodingClient := geocoding.NewClient(httpClient) // создаем клиента для получекния коорлинат
	openmeteoClient := openmeteo.NewClient(httpClient) // создаем клиента для получения температуры

	j, err := sheduler.NewJob( // j содержит job`у (новое задание в планировщике)
		gocron.DurationJob(
			10*time.Second, // задаем интервал срабатывания
		),
		gocron.NewTask(
			func() {
				geoResp, err := geocodingClient.GetCoords(city) // эндпоинт для получения координат
				if err != nil {
					log.Println(err)
					return
				}

				openMeteoResp, err := openmeteoClient.GetTemperature(geoResp.Latitude, geoResp.Longitude)
				if err != nil {
					log.Println(err)
					return
				}

				timestamp, err := time.Parse("2006-01-02T15:04", openMeteoResp.Current.Time)
				if err != nil {
					log.Println(err)
				}

				_, err = conn.Exec(
					ctx,
					"insert into reading (name, timestamp, temperature) values ($1, $2, $3)",
					city, timestamp, openMeteoResp.Current.Temperature2m,
				)

				if err != nil {
					log.Println(err)
					return
				}

				fmt.Printf("%v Update data for city: %s\n", time.Now(), city)
			},
		),
	)
	if err != nil {
		return nil, err
	}

	return []gocron.Job{j}, nil
}
