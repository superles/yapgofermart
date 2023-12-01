package accrual

type ClientMockResponse struct {
	Accrual Accrual // ответ
	Error   error   // ошибка
}

func NewMockClient(rules map[string][]ClientMockResponse) Client {
	return clientMock{rules}
}

type clientMock struct {
	rules map[string][]ClientMockResponse //правила ответов на запросы [номерзаказа][]Ответы
}

func (c clientMock) Get(number string) (Accrual, error) {

	rules, ok := c.rules[number]

	if !ok || len(rules) == 0 {
		return Accrual{}, ErrNotRegistered
	}

	var response = rules[0]

	if len(rules) > 1 {
		c.rules[number] = rules[1:]
	} else {
		c.rules[number] = []ClientMockResponse{}
	}

	if response.Error != nil {
		return Accrual{}, response.Error
	} else {
		return response.Accrual, nil
	}
}
