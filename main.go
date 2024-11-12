package main

import (
    "context"
    "fmt"
    "io/ioutil"
    "net/http"
    "os"
    "sort"
    "strings"
    "time"

    "github.com/joho/godotenv"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
    "go.mongodb.org/mongo-driver/mongo/writeconcern"
)

// Структура для хранения результатов запроса
type PingResult struct {
    Ip         string  // Публичный IP ноды
    DurationMs float64 // Время запроса в миллисекундах с 4 знаками после запятой
    Timestamp  int64   // Время отправки в БД
    Lang       string  // Язык скрипта
}

// Функция для вычисления медианы
func median(arr []float64) float64 {
    sort.Slice(arr, func(i, j int) bool {
        return arr[i] < arr[j]
    })
    length := len(arr)
    if length%2 == 0 {
        return (arr[length/2-1] + arr[length/2]) / 2
    }
    return arr[length/2]
}

// Функция для вычисления среднего значения
func mean(arr []float64) float64 {
    var sum float64
    for _, value := range arr {
        sum += value
    }
    return sum / float64(len(arr))
}

// Функция для получения публичного IP-адреса
func getPublicIP() (string, error) {
    resp, err := http.Get("http://checkip.amazonaws.com/")
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return "", err
    }

    return strings.TrimSpace(string(body)), nil
}

// Функция для подключения к MongoDB
func connectMongoDB() (*mongo.Client, *mongo.Collection, error) {
    // Загружаем данные из .env файла
    if err := godotenv.Load(); err != nil {
        fmt.Println("Ошибка загрузки .env файла:", err)
        return nil, nil, err
    }

    // Чтение переменных из окружения
    mongoURI := os.Getenv("MONGO_URI")
    dbName := os.Getenv("MONGO_DB")
    collectionName := os.Getenv("MONGO_COLLECTION")

    if mongoURI == "" || dbName == "" || collectionName == "" {
        fmt.Println("Переменные окружения MongoDB не установлены, результаты будут выведены только в терминал.")
        return nil, nil, nil
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    clientOptions := options.Client().ApplyURI(mongoURI).SetWriteConcern(writeconcern.Majority())
    client, err := mongo.Connect(ctx, clientOptions)
    if err != nil {
        return nil, nil, err
    }

    collection := client.Database(dbName).Collection(collectionName)
    return client, collection, nil
}

// Основная функция
func main() {
    repeats := 10 // Количество повторов
    var times []float64

    // Подключаемся к MongoDB
    client, collection, err := connectMongoDB()
    if err != nil {
        fmt.Println("Ошибка подключения к MongoDB:", err)
        return
    }
    if client != nil {
        defer client.Disconnect(context.Background())
    }

    // Получаем публичный IP-адрес ноды
    ip, err := getPublicIP()
    if err != nil {
        fmt.Println("Ошибка получения публичного IP:", err)
        return
    }

    // Значение языка для записи в БД
    lang := "go"

    for i := 0; i < repeats; i++ {
        start := time.Now()

        // Выполняем HTTP-запрос
        resp, err := http.Get("https://api.bybit.com/v2/public/time")
        if err != nil {
            fmt.Printf("Request %d failed: %v\n", i+1, err)
            continue
        }
        resp.Body.Close()

        duration := float64(time.Since(start).Microseconds()) / 1000.0 // Время в миллисекундах с 4 знаками
        times = append(times, duration)

        result := PingResult{
            Ip:         ip,
            DurationMs: duration,
            Timestamp:  time.Now().Unix(),
            Lang:       lang,
        }

        if collection != nil {
            _, err = collection.InsertOne(context.Background(), result)
            if err != nil {
                fmt.Printf("Ошибка записи в MongoDB для запроса %d: %v\n", i+1, err)
                continue
            }
        } else {
            fmt.Printf("Request %d: IP=%s, Duration=%.4f ms, Timestamp=%d, Lang=%s\n", i+1, ip, duration, result.Timestamp, lang)
        }

        fmt.Printf("Request %d time: %.4f ms\n", i+1, duration)
    }

    minTime, maxTime := times[0], times[0]
    for _, t := range times {
        if t < minTime {
            minTime = t
        }
        if t > maxTime {
            maxTime = t
        }
    }
    medianTime := median(times)
    meanTime := mean(times)

    fmt.Printf("\nResults over %d requests:\n", repeats)
    fmt.Printf("Min time: %.4f ms\n", minTime)
    fmt.Printf("Max time: %.4f ms\n", maxTime)
    fmt.Printf("Median time: %.4f ms\n", medianTime)
    fmt.Printf("Mean time: %.4f ms\n", meanTime)
}
