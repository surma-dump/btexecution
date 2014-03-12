package main

type CursorMock struct {
	MoreFunc  func() bool
	NextFunc  func(v interface{}) error
	CloseFunc func()
}

func (c *CursorMock) More() bool {
	if c.MoreFunc == nil {
		return false
	}
	return c.MoreFunc()
}

func (c *CursorMock) Next(v interface{}) error {
	if c.NextFunc == nil {
		return nil
	}
	return c.NextFunc(v)
}

func (c *CursorMock) Close() {
	if c.CloseFunc == nil {
		return
	}
	c.CloseFunc()
}
