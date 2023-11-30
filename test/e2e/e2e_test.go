package e2e_test

import (
	"context"
	"fmt"
	"github.com/superles/yapgofermart/internal/accrual"
	"github.com/superles/yapgofermart/internal/config"
	"github.com/superles/yapgofermart/internal/model"
	"github.com/superles/yapgofermart/internal/server"
	"github.com/superles/yapgofermart/internal/storage"
	"github.com/superles/yapgofermart/internal/storage/memstorage"
	"github.com/superles/yapgofermart/internal/utils/logger"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type clientTest struct {
}

func (c clientTest) Get(number string) (accrual.Accrual, error) {
	sum := float64(100)
	return accrual.Accrual{Number: number, Status: accrual.StatusProcessed, Accrual: &sum}, nil
}

type APITestSuite struct {
	suite.Suite
	appContext context.Context
	done       context.CancelFunc
	server     *server.Server
	service    accrual.Service
	storage    storage.Storage
	testUsers  []model.User
	config     *config.Config
}

func (suite *APITestSuite) getFirstUser() model.User {
	if len(suite.testUsers) < 2 {
		suite.FailNow("тестовые пользователи не зарегистрированы")
	}
	return suite.testUsers[0]
}

func (suite *APITestSuite) getSecondUser() model.User {
	if len(suite.testUsers) < 2 {
		suite.FailNow("тестовые пользователи не зарегистрированы")
	}
	return suite.testUsers[1]
}

func (suite *APITestSuite) registerTestUsers() {
	type testUser struct {
		Name     string
		Password string
	}

	users := []testUser{
		{Name: "user", Password: "pass"},
		{Name: "user1", Password: "pass1"},
	}
	var returnUsers []model.User
	for _, user := range users {
		hashPass, err := server.HashPasswordWithRandomSalt(user.Password)
		suite.Require().NoError(err, "ошибка хеширования пароля")
		registerUser, err := suite.storage.RegisterUser(suite.appContext, model.User{Name: user.Name, PasswordHash: hashPass})
		suite.Require().NoError(err, "регистрации пользователя")
		returnUsers = append(returnUsers, registerUser)
	}
	suite.testUsers = returnUsers
}

func (suite *APITestSuite) initConfig() {
	var err error
	suite.config, err = config.New()
	suite.Require().NoError(err, "ошибка инициализации конфига")
	suite.config.LogLevel = "error"
	if len(suite.config.Endpoint) == 0 {
		suite.config.Endpoint = "localhost:33190"
	}
	if len(suite.config.AccrualSystemAddress) == 0 {
		suite.config.AccrualSystemAddress = "localhost:33191"
	}
	if len(suite.config.SecretKey) == 0 {
		suite.config.SecretKey = "test"
		suite.config.SecretKeyBytes = []byte(suite.config.SecretKey)
	}
}

func (suite *APITestSuite) initLogger() {
	err := logger.Initialize(suite.config.LogLevel)
	suite.Require().NoError(err, "ошибка инициализации logger")
}

func (suite *APITestSuite) initStorage() {
	var err error
	suite.storage, err = memstorage.NewStorage()
	suite.Require().NoError(err, "ошибка инициализации бд")
}

func (suite *APITestSuite) initService() {
	suite.service = accrual.Service{Client: clientTest{}, Storage: suite.storage, PoolInterval: 100 * time.Millisecond}
}

func (suite *APITestSuite) initServer() {
	suite.server = server.New(suite.config, suite.storage, suite.service)
}

func (suite *APITestSuite) getAuthHeader(user model.User) string {
	var err error
	var token string
	token, err = suite.server.GetAuthToken(user)
	suite.Require().NoError(err, "ошибка добавления тестового токена user")
	return fmt.Sprintf("Bearer %s", token)
}

func (suite *APITestSuite) SetupTest() {
	var err error
	suite.appContext, suite.done = context.WithCancel(context.Background())
	suite.initConfig()
	suite.initLogger()
	suite.initStorage()
	suite.registerTestUsers()
	suite.initService()
	suite.initServer()
	go func() {
		err = suite.server.Run(suite.appContext)
		suite.Require().NoError(err, "ошибка инициализации сервера")
	}()
	time.Sleep(100 * time.Millisecond) // Как правильно ожидать запуск http сервера
}

func (suite *APITestSuite) TearDownTest() {
	suite.done()
}

// Теста для роута /api/user/register
func (suite *APITestSuite) TestUserRegister() {
	client := http.Client{}
	body := strings.NewReader(`{ "login":"user3", "password":"pass3" }`)
	url := fmt.Sprintf("http://%s%s", suite.config.Endpoint, "/api/user/register")
	req, err := http.NewRequest(http.MethodPost, url, body)
	suite.Require().NoError(err, "ошибка создания запроса")
	req.Header.Set("Content-Type", "application/json")
	res, err := client.Do(req)
	suite.Require().NoError(err, "ошибка запроса")
	defer res.Body.Close()
	suite.Require().Exactlyf(http.StatusOK, res.StatusCode, "неожиданный статус")
	_, err = suite.storage.GetUserByName(suite.appContext, "user")
	suite.Require().NoError(err, "ошибка проверки наличия пользователя в бд")

}

// Теста для роута /api/user/login
func (suite *APITestSuite) TestUserLogin() {
	client := http.Client{}
	body := strings.NewReader(`{ "login":"user", "password":"pass" }`)
	url := fmt.Sprintf("http://%s%s", suite.config.Endpoint, "/api/user/login")
	req, err := http.NewRequest(http.MethodPost, url, body)
	suite.Require().NoError(err, "ошибка создания запроса")
	req.Header.Set("Content-Type", "application/json")
	res, err := client.Do(req)
	suite.Require().NoError(err, "ошибка запроса")
	defer res.Body.Close()
	suite.Require().Exactlyf(http.StatusOK, res.StatusCode, "неожиданный статус")
	_, err = suite.storage.GetUserByName(suite.appContext, "user")
	suite.Require().NoError(err, "ошибка проверки наличия пользователя в бд")

}

func (suite *APITestSuite) TestCreateOrder() {
	client := http.Client{}
	orderNumber := `123456789049`
	body := strings.NewReader(orderNumber)
	url := fmt.Sprintf("http://%s%s", suite.config.Endpoint, "/api/user/orders")
	req, err := http.NewRequest(http.MethodPost, url, body)
	suite.Require().NoError(err, "ошибка создания запроса")
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Authorization", suite.getAuthHeader(suite.getFirstUser()))
	res, err := client.Do(req)
	suite.Require().NoError(err, "ошибка запроса")
	defer res.Body.Close()
	suite.Require().Exactlyf(http.StatusAccepted, res.StatusCode, "неожиданный статус")
	_, err = suite.storage.GetOrder(suite.appContext, orderNumber)
	suite.Require().NoError(err, "ошибка проверки наличия заказа в бд")

}

func (suite *APITestSuite) TestBalanceChangedAfterOrderCreate() {
	client := http.Client{}
	orderNumber := `123456789049`
	body := strings.NewReader(orderNumber)
	url := fmt.Sprintf("http://%s%s", suite.config.Endpoint, "/api/user/orders")
	req, err := http.NewRequest(http.MethodPost, url, body)
	suite.Require().NoError(err, "ошибка создания запроса")
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Authorization", suite.getAuthHeader(suite.getFirstUser()))
	res, err := client.Do(req)
	suite.Require().NoError(err, "ошибка запроса")
	defer res.Body.Close()
	suite.Require().Exactlyf(http.StatusAccepted, res.StatusCode, "неожиданный статус")
	var order model.Order
	order, err = suite.storage.GetOrder(suite.appContext, orderNumber)
	suite.Require().NoError(err, "ошибка проверки наличия пользователя в бд")
	time.Sleep(1 * time.Second) // ожидание обработки сервиса
	var orderOwner model.User
	orderOwner, err = suite.storage.GetUserByID(suite.appContext, order.UserID)
	suite.Require().NoError(err, "ошибка проверки наличия заказа в бд")
	suite.Require().Exactlyf(float64(100), orderOwner.Balance, "неправильно начисленный баланс")

}

func (suite *APITestSuite) TestBalanceChangedAfterOrderAndWithdraw() {
	suite.TestBalanceChangedAfterOrderCreate()
	client := http.Client{}
	bodyString := `{ "order": "2377225624", "sum": 50 }`
	body := strings.NewReader(bodyString)
	url := fmt.Sprintf("http://%s%s", suite.config.Endpoint, "/api/user/balance/withdraw")
	req, err := http.NewRequest(http.MethodPost, url, body)
	suite.Require().NoError(err, "ошибка создания запроса")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", suite.getAuthHeader(suite.getFirstUser()))
	res, err := client.Do(req)
	suite.Require().NoError(err, "ошибка запроса")
	defer res.Body.Close()
	suite.Require().Exactlyf(http.StatusOK, res.StatusCode, "неожиданный статус")
	var orderOwner model.User
	orderOwner, err = suite.storage.GetUserByID(suite.appContext, suite.getFirstUser().ID)
	suite.Require().NoError(err, "ошибка проверки наличия пользователя в бд")
	suite.Require().Exactlyf(float64(50), orderOwner.Balance, "неправильно начисленный баланс")

}

func TestAPI(t *testing.T) {
	suite.Run(t, new(APITestSuite))
}
