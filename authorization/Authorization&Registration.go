// Нужно создать переменную с текущим значением: ошибка или не ошибка. Если прав недостаточно ИЛИ вход выполнен неуспешно, будет ОШИБКА
// Нужно подключить файл с информацией о пользователях
// Нужно создать middleware, состоящий из 3-х слоёв:
//		1) Логирование
//		2) ЕСЛИ (отправлен запрос на вход) -> Вход в систему/регистрация. ИНАЧЕ вниз.
//		3) Проверка прав пользователя

package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

var Users = GetData()
var DataBasePath = "DataBase.txt"
var user User

type UserDataGitHub struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

type UserData struct {
	Name  string
	Group string
}

type User struct {
	GitHub_id   int
	Telegram_id int
	MyToken     string // ЗАПОЛНИТЬ ПОЛЕ MyToken
	Roles       [3]string
	Data        UserData
}

type Authanticate struct {
	//is_done string     // МОЖЕТ ПРИДЁТСЯ РАСКОММЕНТИРОВАТЬ
	code string
}

const (
	CLIENT_ID     = "06f163e42c9edf8bc050"
	CLIENT_SECRET = "2dea7b2aff32ced454b3140fa3df5355755842b1"
)

func main() {
	router := http.NewServeMux()

	// Регистрируем маршруты
	router.HandleFunc("/Oauth", logging(aouth1Hendler))
	router.HandleFunc("/Oauth/redirect", logging(oauthHandler))

	http.ListenAndServe(":8080", router)
}

//func rightsHendler(w http.ResponseWriter, r *http.Request) {
// Фиксируем chat_id и генерируем ссылку
// Последнюю помещаем в wr
//}

func aouth1Hendler(w http.ResponseWriter, r *http.Request) {
	// Получаем Telegram_id и генерируем ссылку +
	user.Telegram_id, _ = strconv.Atoi(r.URL.Query().Get("chat_id")) // От бота поступает Get-запрос!
	authorizationURL := "https://github.com/login/oauth/authorize?client_id=" + CLIENT_ID

	w.Write([]byte(authorizationURL))
}

func oauthHandler(w http.ResponseWriter, r *http.Request) {
	// Не выполнено или требует проверки:
	// Перенаправить пользователя на страницу с оповещение об успешном входе в систему +(?)
	// Сохраняем свой токен в учётку пользователя
	var authanticate Authanticate
	var responceHtml = "<html><body><h1>Вы НЕ аутентифецированы!</h1></body></html>"

	code := r.URL.Query().Get("code")
	if code != "" {
		//authanticate.is_done = true
		authanticate.code = code
		responceHtml = "<html><body><h1>Вы аутентифецированы!</h1></body></html>"
	}
	fmt.Fprint(w, responceHtml)

	accessToken, err := getAccessToken(code)
	if err != nil {
		w.Write([]byte("Ошибка при запросе токена доступа!"))
	}

	userDataGitHub, err := getUserData(accessToken)
	if err != nil {
		w.Write([]byte("Ошибка при запросе данных пользователя!"))
	}
	user.GitHub_id = userDataGitHub.Id
	user.Data.Name = userDataGitHub.Name

	// Проверяем зарегистрирован ли пользователь
	Users = NewUser(&Users, &user)
	SafeData(&Users)

	// Переводим user.GitHub_id в []byte
	b := make([]byte, 8) // 8, потому что GitHub_id имеет тип int64
	binary.LittleEndian.PutUint64(b, uint64(user.GitHub_id))

	w.Write(b)
}

func logging(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("Получен запрос на " + r.URL.Path) // Переделать так, чтобы оно сохраняло логи
		next(w, r)
		log.Println("Отправлен ответ на " + r.URL.Path) // Переделать так, чтобы оно сохраняло логи
	}
}

func getAccessToken(code string) (string, error) {
	// Создаёт http-клиент с дефолтными настройками
	client := http.Client{}
	requestURL := "https://github.com/login/oauth/acces_token"

	// Добавляем данные в виде формы
	form := url.Values{}
	form.Add("client_id", CLIENT_ID)
	form.Add("client_secret", CLIENT_SECRET)
	form.Add("code", code)

	// Готовим и отправляем запрос
	request, _ := http.NewRequest("POST", requestURL, strings.NewReader(form.Encode()))
	request.Header.Set("Accept", "application/json") // Просим прислать ответ в формате json
	responce, err := client.Do(request)

	if err != nil {
		return "", err
	}

	defer responce.Body.Close()

	// Достаём данные из тела овтета
	var responceJSON struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		Scope       string `json:"scope"` // !!!!!!!!Понять что из этого "свой токен"!!!!!!!!!!!!
	}
	json.NewDecoder(responce.Body).Decode(&responceJSON)

	return responceJSON.AccessToken, nil
}

func getUserData(AccessToken string) (UserDataGitHub, error) {
	// Создаём http-клиент с дефольтными настройками
	client := http.Client{}
	requestURL := "https://api.github.com/user"

	var data UserDataGitHub

	// Готовим и отправляем запрос
	request, _ := http.NewRequest("GET", requestURL, nil)
	request.Header.Set("Authorization", "Bearer"+AccessToken)
	responce, err := client.Do(request)
	if err != nil {
		return data, err
	}

	defer responce.Body.Close()
	json.NewDecoder(responce.Body).Decode(&data)

	return data, nil
}

func GetData() map[int]User {

	fileData, err := os.ReadFile(DataBasePath)

	defer func() {
		panicValue := recover()
		if panicValue != nil {
			fmt.Println(panicValue)
		}
	}()

	if err != nil {
		panic(err)
	}

	Data := string(fileData)

	Users := make(map[int]User, len(strings.Split(Data, "\n"))-1)

	for _, UserInfo := range strings.Split(Data, "\n") {
		if len(UserInfo) > 6 {
			Info := strings.Split(UserInfo, " ")

			if len(Info) < 7 {
				continue
			}

			GitHub, err1 := strconv.Atoi(Info[0])
			Telegram, err2 := strconv.Atoi(Info[1])
			if err1 != nil || err2 != nil {
				fmt.Println("Error")
			}

			userData := UserData{
				Name:  Info[5],
				Group: Info[6],
			}

			SignedUser := User{
				GitHub_id:   GitHub,
				Telegram_id: Telegram,
				Roles:       [3]string{Info[2], Info[3], Info[4]},
				Data:        userData,
			}
			Users[GitHub] = SignedUser
		}
	}

	return Users
}

// Что из трёх возвращаемых по code значений — свой токен? И как его нужно использовать?

func NewUser(Users *map[int]User, NewUser *User) map[int]User {
	_, ok := (*Users)[(*NewUser).GitHub_id]
	if ok {
		//fmt.Println("Такой пользователь уже есть")
		return *Users
	}
	NewUsers := make(map[int]User, len(*Users)+1)
	for GitHub_id, user := range *Users {
		NewUsers[GitHub_id] = user
	}
	NewUsers[NewUser.GitHub_id] = *NewUser
	return NewUsers
}

func SafeData(Users *map[int]User) {
	var Data string
	for _, UserInfo := range *Users {

		GitHub := strconv.Itoa(UserInfo.GitHub_id)
		Telegram := strconv.Itoa(UserInfo.Telegram_id)

		Data += GitHub + " " +
			Telegram + " " +
			UserInfo.Roles[0] + " " +
			UserInfo.Roles[1] + " " +
			UserInfo.Roles[2] + " " +
			UserInfo.Data.Name + " " +
			UserInfo.Data.Group + " \n"

	}

	file, err := os.Create(DataBasePath)

	defer func() {
		panicValue := recover()
		if panicValue != nil {
			fmt.Println(panicValue)
		}
	}()

	if err != nil {
		panic(err)
	}

	defer file.Close()

	file.WriteString(Data)
}
