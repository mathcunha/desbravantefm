package main

import (
	"testing"
)

func TestLoadTitle(t *testing.T) {
	body, _ := getPageBody("jose-simao")
	strBody := string(body)
	t.Logf("%v", loadTitle(strBody))
}

func TestGetPageBody(t *testing.T) {
	body, _ := getPageBody("jose-simao")
	strBody := string(body)
	begin, end := getIndexes(strBody)
	//t.Logf("%v \n error:%v", strBody[begin:end], err)
	t.Logf("%v", loadItens(strBody[begin:end]))
}
