package tests

func (d *Account) GetName() string {
	return d.state.Name
}

func (d *Account) GetBalance() float64 {
	return d.state.Balance
}
