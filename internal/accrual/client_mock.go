package accrual

type ClientMockResponse struct {
	Accrual Accrual // ответ
	Error   error   // ошибка
}

type ClientMock struct {
	Rules map[string][]ClientMockResponse //Rules правила ответов на запросы [номерзаказа][]Ответы
}

func (c ClientMock) Get(number string) (Accrual, error) {

	rules, ok := c.Rules[number]

	if !ok || len(rules) == 0 {
		return Accrual{}, ErrNotRegistered
	}

	var response = rules[0]

	if len(rules) > 1 {
		c.Rules[number] = rules[1:]
	} else {
		c.Rules[number] = []ClientMockResponse{}
	}

	if response.Error != nil {
		return Accrual{}, response.Error
	} else {
		return response.Accrual, nil
	}
}
