package main

import (
	"bufio"
	"fmt"
	"gopkg.in/h2non/gentleman.v2"
	body2 "gopkg.in/h2non/gentleman.v2/plugins/body"
	"net/http"
	"os"
	"strings"
)

func main() {
	endpoint := "http://localhost:8081/"
	// контейнер данных для запроса
	// приглашение в консоли
	fmt.Println("Введите длинный URL")
	// открываем потоковое чтение из консоли
	reader := bufio.NewReader(os.Stdin)
	// читаем строку из консоли
	long, err := reader.ReadString('\n')
	if err != nil {
		panic(err)
	}
	long = strings.TrimSuffix(long, "\n")
	// добавляем HTTP-клиент
	client := gentleman.New()
	client.URL(endpoint)

	// пишем запрос
	// запрос методом POST должен, помимо заголовков, содержать тело
	// тело должно быть источником потокового чтения io.Reader
	request := client.Request().
		Method(http.MethodPost).
		SetHeader("Content-Type", "text/plain").
		Use(body2.Reader(strings.NewReader(long)))
	response, err := request.Send()
	if err != nil {
		panic(err)
	}
	// выводим код ответа
	fmt.Println("Статус-код ", response.StatusCode)
	// читаем поток из тела ответа
	// и печатаем его
	fmt.Println(response.String())
}
