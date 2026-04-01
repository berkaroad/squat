package tests

func (a *Account) GetName() string {
	return a.state.Name
}

func (a *Account) GetBalance() float64 {
	return a.state.Balance
}
