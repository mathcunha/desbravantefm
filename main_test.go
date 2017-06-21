package main

import (
	"testing"
)

func TestGetPageBody(t *testing.T) {
	body, _ := getPageBody("jose-simao")
	strBody := string(body)
	begin, end := getIndexes(strBody)
	//t.Logf("%v \n error:%v", strBody[begin:end], err)
	t.Logf("%v", loadItens(strBody[begin:end]))
}
