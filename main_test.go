package main

import (
	"testing"
)

func TestLoadTitle(t *testing.T) {
	body, _ := getPageBody("jose-simao")
	t.Logf("%v", loadTitle(body))
}

func TestGetPageBody(t *testing.T) {
	body, err := getPageBody("jose-simao")
	begin, end := getIndexes(t1, body)
	t.Logf("%d, %d \n error:%v", begin, end, err)
}

func TestBuildReadme(t *testing.T) {
	buildReadme()
}
